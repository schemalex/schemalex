package schemalex_test

import (
	"bytes"
	"flag"
	"io/ioutil"
	"os"
	"testing"

	"github.com/pmezard/go-difflib/difflib"
	"github.com/schemalex/schemalex"
	"github.com/schemalex/schemalex/format"
	"github.com/stretchr/testify/assert"
)

var testFile = ""

func init() {
	flag.StringVar(&testFile, "test-file", testFile, "path to test file")
}

type Spec struct {
	Input  string
	Error  bool
	Expect string
}

func TestParser(t *testing.T) {
	parse := func(title string, spec *Spec) {
		t.Helper()
		t.Run(title, func(t *testing.T) {
			t.Helper()
			testParse(t, spec)
		})
	}

	// create database are ignored
	parse("CreateDatabase", &Spec{
		Input: "create DATABASE hoge",
	})
	parse("CreateDatabaseIfNotExists", &Spec{
		Input: "create DATABASE IF NOT EXISTS hoge",
	})
	parse("CreateDatabase17", &Spec{
		Input: "create DATABASE 17",
		Error: true,
	})
	parse("MultipleCreateDatabase", &Spec{
		Input: "create DATABASE hoge; create database fuga;",
	})
	parse("CreateTableIntegerNoWidth", &Spec{
		Input:  "create table hoge_table ( id integer unsigned not null)",
		Expect: "CREATE TABLE `hoge_table` (\n`id` INT (10) UNSIGNED NOT NULL\n)",
	})
	parse("CStyleComment", &Spec{
		Input:  "create table hoge ( /* id integer unsigned not null */ c varchar not null )",
		Expect: "CREATE TABLE `hoge` (\n`c` VARCHAR NOT NULL\n)",
	})
	parse("DoubleDashComment", &Spec{
		Input:  "create table hoge ( -- id integer unsigned not null;\n c varchar not null )",
		Expect: "CREATE TABLE `hoge` (\n`c` VARCHAR NOT NULL\n)",
	})
	parse("TrailingComma", &Spec{
		Input: `create table hoge (
a varchar(20) default "hoge",
b varchar(20) default 'hoge',
c int not null default 10,
);
`,
		Error: true,
	})
	parse("VariousDefaultTypes", &Spec{
		Input: `create table hoge (
a varchar(20) default "hoge",
b varchar(20) default 'hoge',
c int not null default 10
);
`,
		Expect: "CREATE TABLE `hoge` (\n`a` VARCHAR (20) DEFAULT 'hoge',\n`b` VARCHAR (20) DEFAULT 'hoge',\n`c` INT (11) NOT NULL DEFAULT 10\n)",
	})

	parse("WithPrimaryKey", &Spec{
		Input: `create table hoge (
id bigint unsigned not null auto_increment,
c varchar(20) not null default "hoge",
primary key (id, c)
);
`,
		Expect: "CREATE TABLE `hoge` (\n`id` BIGINT (20) UNSIGNED NOT NULL AUTO_INCREMENT,\n`c` VARCHAR (20) NOT NULL DEFAULT 'hoge',\nPRIMARY KEY (`id`, `c`)\n)",
	})

	parse("WithTableOptions", &Spec{
		Input:  "create table hoge (id bigint unsigned not null auto_increment) ENGINE=InnoDB AUTO_INCREMENT 10 DEFAULT CHARACTER SET = utf8 COMMENT = 'hoge comment';",
		Expect: "CREATE TABLE `hoge` (\n`id` BIGINT (20) UNSIGNED NOT NULL AUTO_INCREMENT\n) ENGINE = InnoDB, AUTO_INCREMENT = 10, DEFAULT CHARACTER SET = utf8, COMMENT = 'hoge comment'",
	})

	parse("NormalizeCharacterSetToCharset", &Spec{
		Input:  "create table hoge (id bigint unsigned not null auto_increment) ENGINE=InnoDB AUTO_INCREMENT 10 DEFAULT CHARSET = utf8 COMMENT = 'hoge comment';",
		Expect: "CREATE TABLE `hoge` (\n`id` BIGINT (20) UNSIGNED NOT NULL AUTO_INCREMENT\n) ENGINE = InnoDB, AUTO_INCREMENT = 10, DEFAULT CHARACTER SET = utf8, COMMENT = 'hoge comment'",
	})
	parse("WithKeyAndIndex", &Spec{
		Input:  "create table hoge ( `id` bigint unsigned not null auto_increment,\n `c` varchar(20) not null,\nKEY (`id`), INDEX (`c`)\n)",
		Expect: "CREATE TABLE `hoge` (\n`id` BIGINT (20) UNSIGNED NOT NULL AUTO_INCREMENT,\n`c` VARCHAR (20) NOT NULL,\nINDEX (`id`),\nINDEX (`c`)\n)",
	})
	parse("WithUniqueKeyPrimaryKey", &Spec{
		Input:  "create table hoge ( `id` bigint unsigned not null auto_increment,\n `c` varchar(20) not null,\nUNIQUE INDEX `uniq_id` (`id`, `c`),\n PRIMARY KEY (`id`)\n )",
		Expect: "CREATE TABLE `hoge` (\n`id` BIGINT (20) UNSIGNED NOT NULL AUTO_INCREMENT,\n`c` VARCHAR (20) NOT NULL,\nUNIQUE INDEX `uniq_id` (`id`, `c`),\nPRIMARY KEY (`id`)\n)",
	})
	parse("WithBsasicForeignKey", &Spec{
		Input:  "create table hoge ( `id` bigint unsigned not null auto_increment,\n `c` varchar(20) not null,\nFOREIGN KEY `fk_c` (`c`) )",
		Expect: "CREATE TABLE `hoge` (\n`id` BIGINT (20) UNSIGNED NOT NULL AUTO_INCREMENT,\n`c` VARCHAR (20) NOT NULL,\nFOREIGN KEY `fk_c` (`c`)\n)",
	})
	parse("WithFulltextIndex1", &Spec{
		Input:  "create table hoge (txt TEXT, fulltext ft_idx(txt))",
		Expect: "CREATE TABLE `hoge` (\n`txt` TEXT,\nFULLTEXT INDEX `ft_idx` (`txt`)\n)",
	})
	parse("WithFulltextIndex2", &Spec{
		Input:  "create table hoge (txt TEXT, fulltext index ft_idx(txt))",
		Expect: "CREATE TABLE `hoge` (\n`txt` TEXT,\nFULLTEXT INDEX `ft_idx` (`txt`)\n)",
	})
	parse("WithFulltextIndex3", &Spec{
		Input:  "create table hoge (txt TEXT, fulltext key ft_idx(txt))",
		Expect: "CREATE TABLE `hoge` (\n`txt` TEXT,\nFULLTEXT INDEX `ft_idx` (`txt`)\n)",
	})
	parse("WithSimpleReferenceForeignKey", &Spec{
		Input:  "create table hoge ( `id` bigint unsigned not null auto_increment,\n `c` varchar(20) not null,\nFOREIGN KEY `fk_c` (`c`) REFERENCES `fuga` (`id`) )",
		Expect: "CREATE TABLE `hoge` (\n`id` BIGINT (20) UNSIGNED NOT NULL AUTO_INCREMENT,\n`c` VARCHAR (20) NOT NULL,\nFOREIGN KEY `fk_c` (`c`) REFERENCES `fuga` (`id`)\n)",
	})
	parse("WithMatchReferenceForeignKey", &Spec{
		Input:  "create table hoge ( `id` bigint unsigned not null auto_increment,\n `c` varchar(20) not null,\nFOREIGN KEY `fk_c` (`c`) REFERENCES `fuga` (`id`) MATCH SIMPLE )",
		Expect: "CREATE TABLE `hoge` (\n`id` BIGINT (20) UNSIGNED NOT NULL AUTO_INCREMENT,\n`c` VARCHAR (20) NOT NULL,\nFOREIGN KEY `fk_c` (`c`) REFERENCES `fuga` (`id`) MATCH SIMPLE\n)",
	})
	parse("WithOnDeleteReferenceForeignKey", &Spec{
		Input:  "create table hoge ( `id` bigint unsigned not null auto_increment,\n `c` varchar(20) not null,\nFOREIGN KEY `fk_c` (`c`) REFERENCES `fuga` (`id`) ON DELETE SET NULL)",
		Expect: "CREATE TABLE `hoge` (\n`id` BIGINT (20) UNSIGNED NOT NULL AUTO_INCREMENT,\n`c` VARCHAR (20) NOT NULL,\nFOREIGN KEY `fk_c` (`c`) REFERENCES `fuga` (`id`) ON DELETE SET NULL\n)",
	})
	parse("WithMatchAndOnDeleteReferenceForeignKey", &Spec{
		Input:  "create table hoge ( `id` bigint unsigned not null auto_increment,\n `c` varchar(20) not null,\nFOREIGN KEY `fk_c` (`c`) REFERENCES `fuga` (`id`) MATCH PARTIAL ON DELETE NO ACTION)",
		Expect: "CREATE TABLE `hoge` (\n`id` BIGINT (20) UNSIGNED NOT NULL AUTO_INCREMENT,\n`c` VARCHAR (20) NOT NULL,\nFOREIGN KEY `fk_c` (`c`) REFERENCES `fuga` (`id`) MATCH PARTIAL ON DELETE NO ACTION\n)",
	})
	parse("WithOnDeleteOnUpdateReferenceForeignKey", &Spec{
		Input:  "create table hoge ( `id` bigint unsigned not null auto_increment,\n `c` varchar(20) not null,\nFOREIGN KEY `fk_c` (`c`) REFERENCES `fuga` (`id`) ON DELETE NO ACTION ON UPDATE CASCADE)",
		Expect: "CREATE TABLE `hoge` (\n`id` BIGINT (20) UNSIGNED NOT NULL AUTO_INCREMENT,\n`c` VARCHAR (20) NOT NULL,\nFOREIGN KEY `fk_c` (`c`) REFERENCES `fuga` (`id`) ON DELETE NO ACTION ON UPDATE CASCADE\n)",
	})
	parse("OnDeleteAfterOnUpdateGotError", &Spec{
		Input: "create table hoge ( `id` bigint unsigned not null auto_increment,\n `c` varchar(20) not null,\nFOREIGN KEY `fk_c` (`c`) REFERENCES `fuga` (`id`) ON UPDATE CASCADE ON DELETE RESTRICT)",
		Error: true,
	})
	parse("UnexpectedIndentShownAfterReferencesFuga", &Spec{
		Input: "create table hoge ( `id` bigint unsigned not null auto_increment,\n `c` varchar(20) not null,\nFOREIGN KEY `fk_c` (`c`) REFERENCES `fuga` (`id`) HOGE )",
		Error: true,
	})
	parse("DecimalNotDefault", &Spec{
		Input:  "create table hoge (`foo` DECIMAL(32,30))",
		Expect: "CREATE TABLE `hoge` (\n`foo` DECIMAL (32,30) DEFAULT NULL\n)",
	})
	parse("UniqueKeyWithConstraint", &Spec{
		Input:  "CREATE TABLE `fuga` ( `id` INTEGER NOT NULL AUTO_INCREMENT, CONSTRAINT `symbol` UNIQUE KEY `uniq_id` USING BTREE (`id`) )",
		Expect: "CREATE TABLE `fuga` (\n`id` INT (11) NOT NULL AUTO_INCREMENT,\nCONSTRAINT `symbol` UNIQUE INDEX `uniq_id` USING BTREE (`id`)\n)",
	})
	parse("DropTableIfExists", &Spec{
		Input:  "DROP TABLE IF EXISTS `konboi_bug`; CREATE TABLE foo(`id` INT)",
		Expect: "CREATE TABLE `foo` (\n`id` INT (11) DEFAULT NULL\n)",
	})
	parse("ColumnCharacterSet", &Spec{
		Input:  "CREATE TABLE `foo` (col TEXT CHARACTER SET latin1)",
		Expect: "CREATE TABLE `foo` (\n`col` TEXT CHARACTER SET `latin1`\n)",
	})
	parse("OnUpdateCurrentTimestampNoDefault", &Spec{
		Input:  "CREATE TABLE `foo` (col DATETIME ON UPDATE CURRENT_TIMESTAMP)",
		Expect: "CREATE TABLE `foo` (\n`col` DATETIME ON UPDATE CURRENT_TIMESTAMP DEFAULT NULL\n)",
	})
	parse("KeyNormalizedToIndex", &Spec{
		Input:  "CREATE TABLE `foo` (col TEXT, KEY col_idx (col(196)))",
		Expect: "CREATE TABLE `foo` (\n`col` TEXT,\nINDEX `col_idx` (`col`(196))\n)",
	})
	parse("CreateTableLike", &Spec{
		Input:  "CREATE TABLE foo LIKE bar",
		Expect: "CREATE TABLE `foo` LIKE `bar`",
	})
	parse("ColumnOptionPrimaryKey", &Spec{
		// see https://github.com/schemalex/schemalex/pull/40
		Input:  "CREATE TABLE foo (id INTEGER PRIMARY KEY AUTO_INCREMENT)",
		Expect: "CREATE TABLE `foo` (\n`id` INT (11) DEFAULT NULL AUTO_INCREMENT,\nPRIMARY KEY (`id`)\n)",
	})
	parse("ColumnOptionCommentPrimaryKey1", &Spec{
		// see https://github.com/schemalex/schemalex/pull/40
		Input:  "CREATE TABLE `test` (\n`id` int(11) PRIMARY KEY COMMENT 'aaa' NOT NULL,\nhoge int default 1 not null COMMENT 'bbb' UNIQUE\n);",
		Expect: "CREATE TABLE `test` (\n`id` INT (11) NOT NULL COMMENT 'aaa',\n`hoge` INT (11) NOT NULL DEFAULT 1 COMMENT 'bbb',\nPRIMARY KEY (`id`),\nUNIQUE INDEX `hoge` (`hoge`)\n)",
	})
	parse("ColumnOptionCommentPrimaryKey2", &Spec{
		// see https://github.com/schemalex/schemalex/pull/40
		Input:  "CREATE TABLE `test` (\n`id` int(11) COMMENT 'aaa' PRIMARY KEY NOT NULL,\nhoge int default 1 UNIQUE not null COMMENT 'bbb'\n);",
		Expect: "CREATE TABLE `test` (\n`id` INT (11) NOT NULL COMMENT 'aaa',\n`hoge` INT (11) NOT NULL DEFAULT 1 COMMENT 'bbb',\nPRIMARY KEY (`id`),\nUNIQUE INDEX `hoge` (`hoge`)\n)",
	})
	parse("Enum", &Spec{
		Input:  "CREATE TABLE `test` (\n`status` ENUM('on', 'off') NOT NULL DEFAULT 'off'\n);",
		Expect: "CREATE TABLE `test` (\n`status` ENUM ('on','off') NOT NULL DEFAULT 'off'\n)",
	})
	parse("Set", &Spec{
		Input:  "CREATE TABLE `test` (\n`status` SET('foo', 'bar', 'baz') NOT NULL DEFAULT 'foo,baz'\n);",
		Expect: "CREATE TABLE `test` (\n`status` SET ('foo','bar','baz') NOT NULL DEFAULT 'foo,baz'\n)",
	})
	parse("BooleanDefaultTrue", &Spec{
		Input:  "CREATE TABLE `test` (\n`valid` BOOLEAN not null default true\n);",
		Expect: "CREATE TABLE `test` (\n`valid` TINYINT (1) NOT NULL DEFAULT 1\n)",
	})
	parse("BooleanDefaultFalse", &Spec{
		Input:  "CREATE TABLE `test` (\n`valid` BOOLEAN not null default false\n);",
		Expect: "CREATE TABLE `test` (\n`valid` TINYINT (1) NOT NULL DEFAULT 0\n)",
	})
	parse("BoolDefaultTrue", &Spec{
		Input:  "CREATE TABLE `test` (\n`valid` BOOL not null default true\n);",
		Expect: "CREATE TABLE `test` (\n`valid` TINYINT (1) NOT NULL DEFAULT 1\n)",
	})
	parse("BoolDefaultFalse", &Spec{
		Input:  "CREATE TABLE `test` (\n`valid` BOOL not null default false\n);",
		Expect: "CREATE TABLE `test` (\n`valid` TINYINT (1) NOT NULL DEFAULT 0\n)",
	})
	parse("JSON", &Spec{
		Input:  "CREATE TABLE `test` (\n`valid` JSON not null\n);",
		Expect: "CREATE TABLE `test` (\n`valid` JSON NOT NULL\n)",
	})
	parse("CreateTableIfNotExists", &Spec{
		Input:  "CREATE TABLE IF NOT EXISTS `test` (\n`id` INT (10) NOT NULL\n);",
		Expect: "CREATE TABLE IF NOT EXISTS `test` (\n`id` INT (10) NOT NULL\n)",
	})
	parse("MultipleTableOptions", &Spec{
		Input:  "CREATE TABLE foo (id INT(10) NOT NULL) ENGINE = InnoDB, DEFAULT CHARACTER SET = utf8mb4",
		Expect: "CREATE TABLE `foo` (\n`id` INT (10) NOT NULL\n) ENGINE = InnoDB, DEFAULT CHARACTER SET = utf8mb4",
	})
	parse("GithubIssue59", &Spec{
		Input: "DROP TABLE IF EXISTS `socialaccount_socialtoken`;\nCREATE TABLE `socialaccount_socialtoken` (\n`id` int(11) NOT NULL AUTO_INCREMENT,\n`token` longtext COLLATE utf8mb4_unicode_ci NOT NULL,\n`token_secret` longtext COLLATE utf8mb4_unicode_ci NOT NULL,\n`expires_at` datetime(6) DEFAULT NULL,\n`account_id` int(11) NOT NULL,\n`app_id` int(11) NOT NULL,\nPRIMARY KEY (`id`) USING BTREE,\nUNIQUE KEY `socialaccount_socialtoken_app_id_account_id_fca4e0ac_uniq` (`app_id`,`account_id`) USING BTREE,\nKEY `socialaccount_social_account_id_951f210e_fk_socialacc` (`account_id`) USING BTREE,\nCONSTRAINT `socialaccount_social_account_id_951f210e_fk_socialacc` FOREIGN KEY (`account_id`) REFERENCES `socialaccount_socialaccount` (`id`),\nCONSTRAINT `socialaccount_social_app_id_636a42d7_fk_socialacc` FOREIGN KEY (`app_id`) REFERENCES `socialaccount_socialapp` (`id`)\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci ROW_FORMAT=DYNAMIC;",
		Expect: "CREATE TABLE `socialaccount_socialtoken` (\n" +
			"`id` INT (11) NOT NULL AUTO_INCREMENT,\n" +
			"`token` LONGTEXT COLLATE `utf8mb4_unicode_ci` NOT NULL,\n" +
			"`token_secret` LONGTEXT COLLATE `utf8mb4_unicode_ci` NOT NULL,\n" +
			"`expires_at` DATETIME (6) DEFAULT NULL,\n" +
			"`account_id` INT (11) NOT NULL,\n" +
			"`app_id` INT (11) NOT NULL,\n" +
			"PRIMARY KEY USING BTREE (`id`),\n" +
			"UNIQUE INDEX `socialaccount_socialtoken_app_id_account_id_fca4e0ac_uniq` USING BTREE (`app_id`, `account_id`),\n" +
			"INDEX `socialaccount_social_account_id_951f210e_fk_socialacc` USING BTREE (`account_id`),\n" +
			"CONSTRAINT `socialaccount_social_account_id_951f210e_fk_socialacc` FOREIGN KEY (`account_id`) REFERENCES `socialaccount_socialaccount` (`id`),\n" +
			"INDEX `socialaccount_social_app_id_636a42d7_fk_socialacc` (`app_id`),\n" +
			"CONSTRAINT `socialaccount_social_app_id_636a42d7_fk_socialacc` FOREIGN KEY (`app_id`) REFERENCES `socialaccount_socialapp` (`id`)\n" +
			") ENGINE = InnoDB, DEFAULT CHARACTER SET = utf8mb4, DEFAULT COLLATE = utf8mb4_unicode_ci, ROW_FORMAT = DYNAMIC",
	})
	parse("CommentsEmptyLines", &Spec{
		Input: `/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;`,
		Expect: "",
	})
	parse("CommentsAndStatementsMixedTogether", &Spec{
		Input:  "/* hello, world*/;\nCREATE TABLE foo (\na int);\n/* hello, world again! */;\nCREATE TABLE bar (\nb int);",
		Expect: "CREATE TABLE `foo` (\n`a` INT (11) DEFAULT NULL\n)CREATE TABLE `bar` (\n`b` INT (11) DEFAULT NULL\n)",
	})
	parse("GithubIssue62", &Spec{
		Input: "DROP TABLE IF EXISTS `some_table`;\r\n" +
			"/*!40101 SET @saved_cs_client     = @@character_set_client */;\r\n" +
			"SET character_set_client = utf8mb4 ;\r\n" +
			"CREATE TABLE `some_table` (\r\n" +
			"  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,\r\n" +
			"  `user_id` varchar(32) DEFAULT NULL,\r\n" +
			"  `context` json DEFAULT NULL,\r\n" +
			"  `created_at` datetime DEFAULT CURRENT_TIMESTAMP,\r\n" +
			"  PRIMARY KEY (`id`),\r\n" +
			"  KEY `created_at` (`created_at` DESC) /*!80000 INVISIBLE */,\r\n" +
			"  KEY `user_id_idx` (`user_id`),\r\n" +
			"  CONSTRAINT `some_table__user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE SET NULL ON UPDATE SET NULL\r\n" +
			") ENGINE=InnoDB AUTO_INCREMENT=19 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;",
		Expect: "CREATE TABLE `some_table` (\n`id` INT (10) UNSIGNED NOT NULL AUTO_INCREMENT,\n`user_id` VARCHAR (32) DEFAULT NULL,\n`context` JSON DEFAULT NULL,\n`created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,\nPRIMARY KEY (`id`),\nINDEX `created_at` (`created_at` DESC),\nINDEX `user_id_idx` (`user_id`),\nINDEX `some_table__user_id` (`user_id`),\nCONSTRAINT `some_table__user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE SET NULL ON UPDATE SET NULL\n) ENGINE = InnoDB, AUTO_INCREMENT = 19, DEFAULT CHARACTER SET = utf8mb4, DEFAULT COLLATE = utf8mb4_0900_ai_ci",
	})
	parse("DefaultNow", &Spec{
		Input:  "create table `test_log` (`created_at` DATETIME default NOW())",
		Expect: "CREATE TABLE `test_log` (\n`created_at` DATETIME DEFAULT NOW()\n)",
	})

	parse("GithubIssue79", &Spec{
		Input: "CREATE TABLE `test_tb` (" +
			"  `t_id` char(17) NOT NULL," +
			"  `t_type` smallint(6) NOT NULL," +
			"  `cur_date` datetime NOT NULL" +
			") ENGINE=InnoDB DEFAULT CHARSET=utf8" +
			"/*!50100 PARTITION BY LIST (`t_type`)" +
			"(PARTITION p_1 VALUES IN (1) ENGINE = InnoDB," +
			" PARTITION p_100 VALUES IN (100) ENGINE = InnoDB) */;" +
			"/*!40101 SET character_set_client = @saved_cs_client */;",
		Expect: "CREATE TABLE `test_tb` (\n`t_id` CHAR (17) NOT NULL,\n`t_type` SMALLINT (6) NOT NULL,\n`cur_date` DATETIME NOT NULL\n) ENGINE = InnoDB, DEFAULT CHARACTER SET = utf8",
	})
	parse("WhiteSpacesBetweenTableOptionsAndSemicolon", &Spec{
		Input:  "CREATE TABLE foo (id INT(10) NOT NULL) ENGINE = InnoDB, DEFAULT CHARACTER SET = utf8mb4 \n/**/ ;",
		Expect: "CREATE TABLE `foo` (\n`id` INT (10) NOT NULL\n) ENGINE = InnoDB, DEFAULT CHARACTER SET = utf8mb4",
	})
}

func testParse(t *testing.T, spec *Spec) {
	t.Helper()

	p := schemalex.New()
	t.Logf("Parsing '%s'", spec.Input)
	stmts, err := p.ParseString(spec.Input)
	if spec.Error {
		if !assert.Error(t, err, "should be an error") {
			return
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
			diff, _ := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
				A:        difflib.SplitLines(spec.Expect),
				B:        difflib.SplitLines(buf.String()),
				FromFile: "Expected",
				ToFile:   "Actual",
				Context:  2,
			})
			t.Logf("%s", diff)
			return
		}
	}
}

func TestFile(t *testing.T) {
	flag.Parse()
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

	expected := "parse error: expected LPAREN at line 2 column 16 (at EOF)\n    \"CREATE TABLE bar\" <---- AROUND HERE"
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
