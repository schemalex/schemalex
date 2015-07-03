package schemalex

import (
	"flag"
	"io/ioutil"
	"strings"
	"testing"
)

var testFile = ""

func init() {
	flag.StringVar(&testFile, "test-file", testFile, "path to test file")
	flag.Parse()
}

func TestParser(t *testing.T) {
	type Spec struct {
		Input  string
		Error  bool
		Expect string
	}

	specs := []Spec{
		// create database are ignored
		{
			Input:  "create DATABASE hoge",
			Error:  false,
			Expect: "",
		},
		{
			Input:  "create DATABASE IF NOT EXISTS hoge",
			Error:  false,
			Expect: "",
		},
		{
			Input:  "create DATABASE 17",
			Error:  true,
			Expect: "",
		},
		{
			Input:  "create DATABASE hoge; create database fuga;",
			Error:  false,
			Expect: "",
		},
		{
			Input:  "create table hoge_table ( id integer unsigned not null)",
			Error:  false,
			Expect: "CREATE TABLE `hoge_table` (\n`id` INTEGER UNSIGNED NOT NULL\n)",
		},
		// UNSIGNED position is wrong
		{
			Input: "create table hoge_table ( id integer not null unsigned)",
			Error: true,
		},
		// with c style comment
		{
			Input:  "create table hoge ( /* id integer unsigned not null */ c varchar not null )",
			Error:  false,
			Expect: "CREATE TABLE `hoge` (\n`c` VARCHAR NOT NULL\n)",
		},
		// with double dash comment
		{
			Input:  "create table hoge ( -- id integer unsigned not null;\n c varchar not null )",
			Error:  false,
			Expect: "CREATE TABLE `hoge` (\n`c` VARCHAR NOT NULL\n)",
		},
		// with primary key
		{
			Input: `create table hoge (
id bigint unsigned not null auto_increment,
c varchar(20) not null default "hoge",
primary key (id, c)
);
`,
			Error:  false,
			Expect: "CREATE TABLE `hoge` (\n`id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,\n`c` VARCHAR (20) NOT NULL DEFAULT \"hoge\",\nPRIMARY KEY (`id`, `c`)\n)",
		},
		// with table options
		{
			Input: `create table hoge (
id bigint unsigned not null auto_increment
) ENGINE=InnoDB AUTO_INCREMENT 10 DEFAULT CHARACTER SET = utf8;
`,
			Error:  false,
			Expect: "CREATE TABLE `hoge` (\n`id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT\n) ENGINE = InnoDB, AUTO_INCREMENT = 10, DEFAULT CHARACTER SET = utf8",
		},
		// with unique key, primary key
		{
			Input:  "create table hoge ( `id` bigint unsigned not null auto_increment,\n `c` varchar(20) not null,\nUNIQUE INDEX `uniq_id` (`id`, `c`),\n PRIMARY KEY (`id`)\n )",
			Error:  false,
			Expect: "CREATE TABLE `hoge` (\n`id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,\n`c` VARCHAR (20) NOT NULL,\nUNIQUE INDEX `uniq_id` (`id`, `c`),\nPRIMARY KEY (`id`)\n)",
		},
		// with basic foreign key
		{
			Input:  "create table hoge ( `id` bigint unsigned not null auto_increment,\n `c` varchar(20) not null,\nFOREIGN KEY `fk_c` (`c`) )",
			Error:  false,
			Expect: "CREATE TABLE `hoge` (\n`id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,\n`c` VARCHAR (20) NOT NULL,\nFOREIGN KEY `fk_c` (`c`)\n)",
		},
	}

	for _, spec := range specs {
		stmts, err := NewParser(spec.Input).Parse()
		if spec.Error {
			if err == nil {
				t.Errorf("should err: input:%v", spec.Input)
				continue
			}
			t.Log("input:", spec.Input, "error:", err)
		} else {
			if err != nil {
				t.Errorf(err.Error())
				t.Logf("input:%s", spec.Input)
				continue
			}

			var strs []string

			for _, stmt := range stmts {
				strs = append(strs, stmt.String())
			}

			if e, g := spec.Expect, strings.Join(strs, ";\n"); e != g {
				t.Errorf("should:%q\n got:%q", e, g)
			}
		}
	}
}

func TestFile(t *testing.T) {
	if testFile == "" {
		t.Skipf("test-file is nil")
		return
	}

	byt, err := ioutil.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}
	stmts, err := NewParser(string(byt)).Parse()
	if err != nil {
		t.Fatal(err)
	}
	for _, stmt := range stmts {
		t.Log(stmt)
	}
}
