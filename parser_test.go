package schemalex

import (
	"bytes"
	"flag"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
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
		// with simple reference foreign key
		{
			Input:  "create table hoge ( `id` bigint unsigned not null auto_increment,\n `c` varchar(20) not null,\nFOREIGN KEY `fk_c` (`c`) REFERENCES `fuga` (`id`) )",
			Error:  false,
			Expect: "CREATE TABLE `hoge` (\n`id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,\n`c` VARCHAR (20) NOT NULL,\nFOREIGN KEY `fk_c` (`c`) REFERENCES `fuga` (`id`)\n)",
		},
		// with match reference foreign key
		{
			Input:  "create table hoge ( `id` bigint unsigned not null auto_increment,\n `c` varchar(20) not null,\nFOREIGN KEY `fk_c` (`c`) REFERENCES `fuga` (`id`) MATCH SIMPLE )",
			Error:  false,
			Expect: "CREATE TABLE `hoge` (\n`id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,\n`c` VARCHAR (20) NOT NULL,\nFOREIGN KEY `fk_c` (`c`) REFERENCES `fuga` (`id`) MATCH SIMPLE\n)",
		},
		// with on delete reference foreign key
		{
			Input:  "create table hoge ( `id` bigint unsigned not null auto_increment,\n `c` varchar(20) not null,\nFOREIGN KEY `fk_c` (`c`) REFERENCES `fuga` (`id`) ON DELETE SET NULL)",
			Error:  false,
			Expect: "CREATE TABLE `hoge` (\n`id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,\n`c` VARCHAR (20) NOT NULL,\nFOREIGN KEY `fk_c` (`c`) REFERENCES `fuga` (`id`) ON DELETE SET NULL\n)",
		},
		// with match and on delete reference foreign key
		{
			Input:  "create table hoge ( `id` bigint unsigned not null auto_increment,\n `c` varchar(20) not null,\nFOREIGN KEY `fk_c` (`c`) REFERENCES `fuga` (`id`) MATCH PARTIAL ON DELETE NO ACTION)",
			Error:  false,
			Expect: "CREATE TABLE `hoge` (\n`id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,\n`c` VARCHAR (20) NOT NULL,\nFOREIGN KEY `fk_c` (`c`) REFERENCES `fuga` (`id`) MATCH PARTIAL ON DELETE NO ACTION\n)",
		},
		// with on delete, on update reference foreign key
		{
			Input:  "create table hoge ( `id` bigint unsigned not null auto_increment,\n `c` varchar(20) not null,\nFOREIGN KEY `fk_c` (`c`) REFERENCES `fuga` (`id`) ON DELETE NO ACTION ON UPDATE CASCADE)",
			Error:  false,
			Expect: "CREATE TABLE `hoge` (\n`id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,\n`c` VARCHAR (20) NOT NULL,\nFOREIGN KEY `fk_c` (`c`) REFERENCES `fuga` (`id`) ON DELETE NO ACTION ON UPDATE CASCADE\n)",
		},
		// on delete after on update got error
		{
			Input:  "create table hoge ( `id` bigint unsigned not null auto_increment,\n `c` varchar(20) not null,\nFOREIGN KEY `fk_c` (`c`) REFERENCES `fuga` (`id`) ON UPDATE CASCADE ON DELETE RESTRICT)",
			Error:  true,
			Expect: "",
		},
		// unexpected ident shown after references `fuga` (`id`)
		{
			Input:  "create table hoge ( `id` bigint unsigned not null auto_increment,\n `c` varchar(20) not null,\nFOREIGN KEY `fk_c` (`c`) REFERENCES `fuga` (`id`) HOGE )",
			Error:  true,
			Expect: "",
		},
	}

	p := New()
	for _, spec := range specs {
		t.Logf("Parsing '%s'", spec.Input)
		stmts, err := p.ParseString(spec.Input)
		if spec.Error {
			if !assert.Error(t, err, "should be an error") {
				continue
			}
		} else {
			if err != nil {
				t.Errorf(err.Error())
				return
			}

			var buf bytes.Buffer
			stmts.WriteTo(&buf)

			if !assert.Equal(t, spec.Expect, buf.String(), "should match") {
				return
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
	stmts, err := New().Parse(byt)
	if err != nil {
		t.Fatal(err)
	}
	for _, stmt := range stmts {
		t.Log(stmt)
	}
}
