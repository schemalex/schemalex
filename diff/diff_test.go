package diff_test

import (
	"bytes"
	"testing"

	"github.com/schemalex/schemalex/diff"
	"github.com/stretchr/testify/assert"
)

func TestDiff(t *testing.T) {
	type Spec struct {
		Before string
		After  string
		Expect string
	}

	specs := []Spec{
		// drop table
		{
			Before: "CREATE TABLE `hoge` ( `id` integer not null ); CREATE TABLE `fuga` ( `id` INTEGER NOT NULL );",
			After:  "CREATE TABLE `fuga` ( `id` INTEGER NOT NULL );",
			Expect: "DROP TABLE `hoge`;",
		},
		// create table
		{
			Before: "CREATE TABLE `fuga` ( `id` INTEGER NOT NULL );",
			After:  "CREATE TABLE `hoge` ( `id` INTEGER NOT NULL ) ENGINE=InnoDB DEFAULT CHARACTER SET utf8mb4 COMMENT 'table comment'; CREATE TABLE `fuga` ( `id` INTEGER NOT NULL );",
			Expect: "CREATE TABLE `hoge` (\n`id` INTEGER NOT NULL\n) ENGINE = InnoDB, DEFAULT CHARACTER SET = utf8mb4, COMMENT = 'table comment';",
		},
		// drop column
		{
			Before: "CREATE TABLE `fuga` ( `id` INTEGER NOT NULL, `c` VARCHAR (20) NOT NULL DEFAULT 'xxx' );",
			After:  "CREATE TABLE `fuga` ( `id` INTEGER NOT NULL );",
			Expect: "ALTER TABLE `fuga` DROP COLUMN `c`;",
		},
		// add column
		{
			Before: "CREATE TABLE `fuga` ( `id` INTEGER NOT NULL );",
			After:  "CREATE TABLE `fuga` ( `id` INTEGER NOT NULL, `c` VARCHAR (20) NOT NULL DEFAULT 'xxx');",
			Expect: "ALTER TABLE `fuga` ADD COLUMN `c` VARCHAR (20) NOT NULL DEFAULT \"xxx\";",
		},
		// change column
		{
			Before: "CREATE TABLE `fuga` ( `id` INTEGER NOT NULL );",
			After:  "CREATE TABLE `fuga` ( `id` BIGINT NOT NULL );",
			Expect: "ALTER TABLE `fuga` CHANGE COLUMN `id` `id` BIGINT NOT NULL;",
		},
		// change column with comment
		{
			Before: "CREATE TABLE `fuga` ( `id` INTEGER NOT NULL );",
			After:  "CREATE TABLE `fuga` ( `id` INTEGER NOT NULL COMMENT 'fuga is good' );",
			Expect: "ALTER TABLE `fuga` CHANGE COLUMN `id` `id` INTEGER NOT NULL COMMENT 'fuga is good';",
		},
		// drop primary key
		{
			Before: "CREATE TABLE `fuga` ( `id` INTEGER NOT NULL AUTO_INCREMENT, PRIMARY KEY (`id`) );",
			After:  "CREATE TABLE `fuga` ( `id` INTEGER NOT NULL AUTO_INCREMENT );",
			Expect: "ALTER TABLE `fuga` DROP INDEX PRIMARY KEY;",
		},
		// add primary key
		{
			Before: "CREATE TABLE `fuga` ( `id` INTEGER NOT NULL AUTO_INCREMENT );",
			After:  "CREATE TABLE `fuga` ( `id` INTEGER NOT NULL AUTO_INCREMENT, PRIMARY KEY (`id`) );",
			Expect: "ALTER TABLE `fuga` ADD PRIMARY KEY (`id`);",
		},
		// drop unique key
		{
			Before: "CREATE TABLE `fuga` ( `id` INTEGER NOT NULL AUTO_INCREMENT, CONSTRAINT `symbol` UNIQUE KEY `uniq_id` USING BTREE (`id`) );",
			After:  "CREATE TABLE `fuga` ( `id` INTEGER NOT NULL AUTO_INCREMENT );",
			Expect: "ALTER TABLE `fuga` DROP INDEX `uniq_id`;",
		},
		// add unique key
		{
			Before: "CREATE TABLE `fuga` ( `id` INTEGER NOT NULL AUTO_INCREMENT );",
			After:  "CREATE TABLE `fuga` ( `id` INTEGER NOT NULL AUTO_INCREMENT, CONSTRAINT `symbol` UNIQUE KEY `uniq_id` USING BTREE (`id`) );",
			Expect: "ALTER TABLE `fuga` ADD CONSTRAINT `symbol` UNIQUE INDEX `uniq_id` USING BTREE (`id`);",
		},
		// not change index
		{
			Before: "CREATE TABLE `fuga` ( `id` INTEGER NOT NULL AUTO_INCREMENT, CONSTRAINT `symbol` UNIQUE KEY `uniq_id` USING BTREE (`id`) );",
			After:  "CREATE TABLE `fuga` ( `id` INTEGER NOT NULL AUTO_INCREMENT, CONSTRAINT `symbol` UNIQUE KEY `uniq_id` USING BTREE (`id`) );",
			Expect: "",
		},
		// not change FOREIGN KEY
		{
			Before: "CREATE TABLE `fuga` ( `id` INTEGER NOT NULL AUTO_INCREMENT, `fid` INTEGER NOT NULL, FOREIGN KEY fk (fid) REFERENCES f (id) );",
			After:  "CREATE TABLE `fuga` ( `id` INTEGER NOT NULL AUTO_INCREMENT, `fid` INTEGER NOT NULL, FOREIGN KEY fk (fid) REFERENCES f (id) );",
			Expect: "",
		},
		// change CONSTRAINT symbol naml
		{
			Before: "CREATE TABLE `fuga` ( `id` INTEGER NOT NULL AUTO_INCREMENT, `fid` INTEGER NOT NULL, CONSTRAINT `fsym` FOREIGN KEY (fid) REFERENCES f (id) );",
			After:  "CREATE TABLE `fuga` ( `id` INTEGER NOT NULL AUTO_INCREMENT, `fid` INTEGER NOT NULL, CONSTRAINT `ksym` FOREIGN KEY (fid) REFERENCES f (id) );",
			Expect: "ALTER TABLE `fuga` DROP FOREIGN KEY `fsym`;\nALTER TABLE `fuga` ADD CONSTRAINT `ksym` FOREIGN KEY (`fid`) REFERENCES `f` (`id`);",
		},
		// remove FOREIGN KEY
		{
			Before: "CREATE TABLE `fuga` ( `id` INTEGER NOT NULL AUTO_INCREMENT, `fid` INTEGER NOT NULL, FOREIGN KEY fk (fid) REFERENCES f (id) );",
			After:  "CREATE TABLE `fuga` ( `id` INTEGER NOT NULL AUTO_INCREMENT, `fid` INTEGER NOT NULL, INDEX fid (fid) );",
			Expect: "ALTER TABLE `fuga` DROP FOREIGN KEY `fk`;\nALTER TABLE `fuga` ADD INDEX `fid` (`fid`);",
		},
		// multi modify
		{
			Before: "CREATE TABLE `fuga` ( `id` INTEGER NOT NULL AUTO_INCREMENT, `aid` INTEGER NOT NULL, `bid` INTEGER NOT NULL, INDEX `ab` (`aid`, `bid`) );",
			After:  "CREATE TABLE `fuga` ( `id` INTEGER NOT NULL AUTO_INCREMENT, `aid` INTEGER NOT NULL, `cid` INTEGER NOT NULL, INDEX `ac` (`aid`, `cid`) );",
			Expect: "ALTER TABLE `fuga` DROP INDEX `ab`;\nALTER TABLE `fuga` DROP COLUMN `bid`;\nALTER TABLE `fuga` ADD COLUMN `cid` INTEGER NOT NULL;\nALTER TABLE `fuga` ADD INDEX `ac` (`aid`, `cid`);",
		},
	}

	var buf bytes.Buffer
	for _, spec := range specs {
		buf.Reset()

		if !assert.NoError(t, diff.Strings(&buf, spec.Before, spec.After), "diff.String should succeed") {
			return
		}
		if !assert.Equal(t, spec.Expect, buf.String(), "result SQL should match") {
			return
		}
	}
}
