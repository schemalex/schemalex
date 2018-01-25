package schemalex_test

import (
	"bytes"
	"flag"
	"io/ioutil"
	"os"
	"testing"

	"github.com/schemalex/schemalex"
	"github.com/schemalex/schemalex/format"
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
			Expect: "CREATE TABLE `hoge_table` (\n`id` INT (10) UNSIGNED NOT NULL\n)",
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
		// trailing comma
		{
			Input: `create table hoge (
a varchar(20) default "hoge",
b varchar(20) default 'hoge',
c int not null default 10,
);
`,
			Error: true,
		},
		// various default types
		{
			Input: `create table hoge (
a varchar(20) default "hoge",
b varchar(20) default 'hoge',
c int not null default 10
);
`,
			Error:  false,
			Expect: "CREATE TABLE `hoge` (\n`a` VARCHAR (20) DEFAULT 'hoge',\n`b` VARCHAR (20) DEFAULT 'hoge',\n`c` INT (11) NOT NULL DEFAULT 10\n)",
		},
		// with primary key
		{
			Input: `create table hoge (
id bigint unsigned not null auto_increment,
c varchar(20) not null default "hoge",
primary key (id, c)
);
`,
			Expect: "CREATE TABLE `hoge` (\n`id` BIGINT (20) UNSIGNED NOT NULL AUTO_INCREMENT,\n`c` VARCHAR (20) NOT NULL DEFAULT 'hoge',\nPRIMARY KEY (`id`, `c`)\n)",
		},
		// with table options
		{
			Input:  "create table hoge (id bigint unsigned not null auto_increment) ENGINE=InnoDB AUTO_INCREMENT 10 DEFAULT CHARACTER SET = utf8 COMMENT = 'hoge comment';",
			Expect: "CREATE TABLE `hoge` (\n`id` BIGINT (20) UNSIGNED NOT NULL AUTO_INCREMENT\n) ENGINE = InnoDB, AUTO_INCREMENT = 10, DEFAULT CHARACTER SET = utf8, COMMENT = 'hoge comment'",
		},
		// CHARACTER SET -> CHARSET
		{
			Input:  "create table hoge (id bigint unsigned not null auto_increment) ENGINE=InnoDB AUTO_INCREMENT 10 DEFAULT CHARSET = utf8 COMMENT = 'hoge comment';",
			Expect: "CREATE TABLE `hoge` (\n`id` BIGINT (20) UNSIGNED NOT NULL AUTO_INCREMENT\n) ENGINE = InnoDB, AUTO_INCREMENT = 10, DEFAULT CHARACTER SET = utf8, COMMENT = 'hoge comment'",
		},
		// with key, index
		{
			Input:  "create table hoge ( `id` bigint unsigned not null auto_increment,\n `c` varchar(20) not null,\nKEY (`id`), INDEX (`c`)\n)",
			Error:  false,
			Expect: "CREATE TABLE `hoge` (\n`id` BIGINT (20) UNSIGNED NOT NULL AUTO_INCREMENT,\n`c` VARCHAR (20) NOT NULL,\nINDEX (`id`),\nINDEX (`c`)\n)",
		},
		// with unique key, primary key
		{
			Input:  "create table hoge ( `id` bigint unsigned not null auto_increment,\n `c` varchar(20) not null,\nUNIQUE INDEX `uniq_id` (`id`, `c`),\n PRIMARY KEY (`id`)\n )",
			Error:  false,
			Expect: "CREATE TABLE `hoge` (\n`id` BIGINT (20) UNSIGNED NOT NULL AUTO_INCREMENT,\n`c` VARCHAR (20) NOT NULL,\nUNIQUE INDEX `uniq_id` (`id`, `c`),\nPRIMARY KEY (`id`)\n)",
		},
		// with basic foreign key
		{
			Input:  "create table hoge ( `id` bigint unsigned not null auto_increment,\n `c` varchar(20) not null,\nFOREIGN KEY `fk_c` (`c`) )",
			Error:  false,
			Expect: "CREATE TABLE `hoge` (\n`id` BIGINT (20) UNSIGNED NOT NULL AUTO_INCREMENT,\n`c` VARCHAR (20) NOT NULL,\nFOREIGN KEY `fk_c` (`c`)\n)",
		},
		// with fulltext index
		{
			Input:  "create table hoge (txt TEXT, fulltext ft_idx(txt))",
			Error:  false,
			Expect: "CREATE TABLE `hoge` (\n`txt` TEXT,\nFULLTEXT INDEX `ft_idx` (`txt`)\n)",
		},
		// with simple reference foreign key
		{
			Input:  "create table hoge ( `id` bigint unsigned not null auto_increment,\n `c` varchar(20) not null,\nFOREIGN KEY `fk_c` (`c`) REFERENCES `fuga` (`id`) )",
			Error:  false,
			Expect: "CREATE TABLE `hoge` (\n`id` BIGINT (20) UNSIGNED NOT NULL AUTO_INCREMENT,\n`c` VARCHAR (20) NOT NULL,\nFOREIGN KEY `fk_c` (`c`) REFERENCES `fuga` (`id`)\n)",
		},
		// with match reference foreign key
		{
			Input:  "create table hoge ( `id` bigint unsigned not null auto_increment,\n `c` varchar(20) not null,\nFOREIGN KEY `fk_c` (`c`) REFERENCES `fuga` (`id`) MATCH SIMPLE )",
			Error:  false,
			Expect: "CREATE TABLE `hoge` (\n`id` BIGINT (20) UNSIGNED NOT NULL AUTO_INCREMENT,\n`c` VARCHAR (20) NOT NULL,\nFOREIGN KEY `fk_c` (`c`) REFERENCES `fuga` (`id`) MATCH SIMPLE\n)",
		},
		// with on delete reference foreign key
		{
			Input:  "create table hoge ( `id` bigint unsigned not null auto_increment,\n `c` varchar(20) not null,\nFOREIGN KEY `fk_c` (`c`) REFERENCES `fuga` (`id`) ON DELETE SET NULL)",
			Error:  false,
			Expect: "CREATE TABLE `hoge` (\n`id` BIGINT (20) UNSIGNED NOT NULL AUTO_INCREMENT,\n`c` VARCHAR (20) NOT NULL,\nFOREIGN KEY `fk_c` (`c`) REFERENCES `fuga` (`id`) ON DELETE SET NULL\n)",
		},
		// with match and on delete reference foreign key
		{
			Input:  "create table hoge ( `id` bigint unsigned not null auto_increment,\n `c` varchar(20) not null,\nFOREIGN KEY `fk_c` (`c`) REFERENCES `fuga` (`id`) MATCH PARTIAL ON DELETE NO ACTION)",
			Error:  false,
			Expect: "CREATE TABLE `hoge` (\n`id` BIGINT (20) UNSIGNED NOT NULL AUTO_INCREMENT,\n`c` VARCHAR (20) NOT NULL,\nFOREIGN KEY `fk_c` (`c`) REFERENCES `fuga` (`id`) MATCH PARTIAL ON DELETE NO ACTION\n)",
		},
		// with on delete, on update reference foreign key
		{
			Input:  "create table hoge ( `id` bigint unsigned not null auto_increment,\n `c` varchar(20) not null,\nFOREIGN KEY `fk_c` (`c`) REFERENCES `fuga` (`id`) ON DELETE NO ACTION ON UPDATE CASCADE)",
			Error:  false,
			Expect: "CREATE TABLE `hoge` (\n`id` BIGINT (20) UNSIGNED NOT NULL AUTO_INCREMENT,\n`c` VARCHAR (20) NOT NULL,\nFOREIGN KEY `fk_c` (`c`) REFERENCES `fuga` (`id`) ON DELETE NO ACTION ON UPDATE CASCADE\n)",
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
		{
			Input:  "create table hoge (`foo` DECIMAL(32,30))",
			Expect: "CREATE TABLE `hoge` (\n`foo` DECIMAL (32,30) DEFAULT NULL\n)",
		},
		{
			Input:  "CREATE TABLE `fuga` ( `id` INTEGER NOT NULL AUTO_INCREMENT, CONSTRAINT `symbol` UNIQUE KEY `uniq_id` USING BTREE (`id`) )",
			Expect: "CREATE TABLE `fuga` (\n`id` INT (11) NOT NULL AUTO_INCREMENT,\nCONSTRAINT `symbol` UNIQUE INDEX `uniq_id` USING BTREE (`id`)\n)",
		},
		{
			Input:  "DROP TABLE IF EXISTS `konboi_bug`; CREATE TABLE foo(`id` INT)",
			Expect: "CREATE TABLE `foo` (\n`id` INT (11) DEFAULT NULL\n)",
		},
		{
			Input:  "CREATE TABLE `foo` (col TEXT CHARACTER SET latin1)",
			Expect: "CREATE TABLE `foo` (\n`col` TEXT CHARACTER SET `latin1`\n)",
		},
		{
			Input:  "CREATE TABLE `foo` (col DATETIME ON UPDATE CURRENT_TIMESTAMP)",
			Expect: "CREATE TABLE `foo` (\n`col` DATETIME ON UPDATE CURRENT_TIMESTAMP DEFAULT NULL\n)",
		},
		{
			Input:  "CREATE TABLE `foo` (col TEXT, KEY col_idx (col(196)))",
			Expect: "CREATE TABLE `foo` (\n`col` TEXT,\nINDEX `col_idx` (`col`(196))\n)",
		},
		{
			Input:  "CREATE TABLE foo LIKE bar",
			Expect: "CREATE TABLE `foo` LIKE `bar`",
		},
		// see https://github.com/schemalex/schemalex/pull/40
		{
			Input:  "CREATE TABLE foo (id INTEGER PRIMARY KEY AUTO_INCREMENT)",
			Expect: "CREATE TABLE `foo` (\n`id` INT (11) DEFAULT NULL AUTO_INCREMENT,\nPRIMARY KEY (`id`)\n)",
		},
		// see https://github.com/schemalex/schemalex/pull/40
		{
			Input:  "CREATE TABLE `test` (\n`id` int(11) PRIMARY KEY COMMENT 'aaa' NOT NULL,\nhoge int default 1 not null COMMENT 'bbb' UNIQUE\n);",
			Expect: "CREATE TABLE `test` (\n`id` INT (11) NOT NULL COMMENT 'aaa',\n`hoge` INT (11) NOT NULL DEFAULT 1 COMMENT 'bbb',\nPRIMARY KEY (`id`),\nUNIQUE INDEX `hoge` (`hoge`)\n)",
		},
		// see https://github.com/schemalex/schemalex/pull/40
		{
			Input:  "CREATE TABLE `test` (\n`id` int(11) COMMENT 'aaa' PRIMARY KEY NOT NULL,\nhoge int default 1 UNIQUE not null COMMENT 'bbb'\n);",
			Expect: "CREATE TABLE `test` (\n`id` INT (11) NOT NULL COMMENT 'aaa',\n`hoge` INT (11) NOT NULL DEFAULT 1 COMMENT 'bbb',\nPRIMARY KEY (`id`),\nUNIQUE INDEX `hoge` (`hoge`)\n)",
		},
		// ENUM
		{
			Input:  "CREATE TABLE `test` (\n`status` ENUM('on', 'off') NOT NULL DEFAULT 'off'\n);",
			Expect: "CREATE TABLE `test` (\n`status` ENUM ('on','off') NOT NULL DEFAULT 'off'\n)",
		},
		// SET
		{
			Input:  "CREATE TABLE `test` (\n`status` SET('foo', 'bar', 'baz') NOT NULL DEFAULT 'foo,baz'\n);",
			Expect: "CREATE TABLE `test` (\n`status` SET ('foo','bar','baz') NOT NULL DEFAULT 'foo,baz'\n)",
		},
		// BOOLEAN
		{
			Input:  "CREATE TABLE `test` (\n`valid` BOOLEAN not null default true\n);",
			Expect: "CREATE TABLE `test` (\n`valid` TINYINT (1) NOT NULL DEFAULT 1\n)",
		},
		{
			Input:  "CREATE TABLE `test` (\n`valid` BOOLEAN not null default false\n);",
			Expect: "CREATE TABLE `test` (\n`valid` TINYINT (1) NOT NULL DEFAULT 0\n)",
		},
		// BOOL
		{
			Input:  "CREATE TABLE `test` (\n`valid` BOOL not null default true\n);",
			Expect: "CREATE TABLE `test` (\n`valid` TINYINT (1) NOT NULL DEFAULT 1\n)",
		},
		{
			Input:  "CREATE TABLE `test` (\n`valid` BOOL not null default false\n);",
			Expect: "CREATE TABLE `test` (\n`valid` TINYINT (1) NOT NULL DEFAULT 0\n)",
		},
		// CREATE TABLE IF NOT EXISTS
		{
			Input:  "CREATE TABLE IF NOT EXISTS `test` (\n`id` INT (10) NOT NULL\n);",
			Expect: "CREATE TABLE IF NOT EXISTS `test` (\n`id` INT (10) NOT NULL\n)",
		},

		// multiple table options
		{
			Input:  "CREATE TABLE foo (id INT(10) NOT NULL) ENGINE = InnoDB, DEFAULT CHARACTER SET = utf8mb4",
			Expect: "CREATE TABLE `foo` (\n`id` INT (10) NOT NULL\n) ENGINE = InnoDB, DEFAULT CHARACTER SET = utf8mb4",
		},
	}

	p := schemalex.New()
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
			if !assert.NoError(t, format.SQL(&buf, stmts), `format.SQL should succeed`) {
				return
			}

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
	stmts, err := schemalex.New().Parse(byt)
	if err != nil {
		t.Fatal(err)
	}
	for _, stmt := range stmts {
		t.Log(stmt)
	}
}

func TestParseError1(t *testing.T) {
	const src = "CREATE TABLE foo (id int PRIMARY KEY);\nCREATE TABLE bar"
	p := schemalex.New()
	_, err := p.ParseString(src)
	if !assert.Error(t, err, "parse should fail") {
		return
	}

	expected := "parse error: expected RPAREN at line 2 column 16 (at EOF)\n    \"CREATE TABLE bar\" <---- AROUND HERE"
	if !assert.Equal(t, expected, err.Error(), "error matches") {
		return
	}
}

func TestParseError2(t *testing.T) {
	const src = "CREATE TABLE foo (id int PRIMARY KEY);\nCREATE TABLE bar (id int PRIMARY KEY baz TEXT)"
	p := schemalex.New()
	_, err := p.ParseString(src)
	if !assert.Error(t, err, "parse should fail") {
		return
	}

	expected := "parse error: unexpected column option IDENT at line 2 column 37\n    \"CREATE TABLE bar (id int PRIMARY KEY \" <---- AROUND HERE"
	if !assert.Equal(t, expected, err.Error(), "error matches") {
		return
	}
}

func TestParseFileError(t *testing.T) {
	f, err := ioutil.TempFile("", "schemalex-file")
	if !assert.NoError(t, err, "creating tempfile should succeed") {
		return
	}
	defer os.Remove(f.Name())
	defer f.Close()

	f.Write([]byte("CREATE TABLE foo (id int PRIMARY KEY);\nCREATE TABLE bar (id int PRIMARY KEY baz TEXT)"))
	f.Sync()

	p := schemalex.New()
	_, err = p.ParseFile(f.Name())
	if !assert.Error(t, err, "schemalex.ParseFile should fail") {
		return
	}

	pe, ok := err.(schemalex.ParseError)
	if !assert.True(t, ok, "err is a ParseError") {
		return
	}

	if !assert.Equal(t, f.Name(), pe.File(), "pe.File() should be the filename") {
		return
	}

	expected := "parse error: unexpected column option IDENT in file " + f.Name() + " at line 2 column 37\n    \"CREATE TABLE bar (id int PRIMARY KEY \" <---- AROUND HERE"
	if !assert.Equal(t, expected, pe.Error(), "pe.Error() matches expected") {
		return
	}
}
