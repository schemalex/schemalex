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
			Expect: "CREATE TABLE `hoge` (\n`id` INT (11) NOT NULL\n) ENGINE = InnoDB, DEFAULT CHARACTER SET = utf8mb4, COMMENT = 'table comment';",
		},
		// drop column
		{
			Before: "CREATE TABLE `fuga` ( `id` INTEGER NOT NULL, `c` VARCHAR (20) NOT NULL DEFAULT 'xxx' );",
			After:  "CREATE TABLE `fuga` ( `id` INTEGER NOT NULL );",
			Expect: "ALTER TABLE `fuga` DROP COLUMN `c`;",
		},
		// add column (after)
		{
			Before: "CREATE TABLE `fuga` ( `id` INTEGER NOT NULL );",
			After:  "CREATE TABLE `fuga` ( `id` INTEGER NOT NULL, `a` INTEGER NOT NULL, `b` INTEGER NOT NULL, `c` INTEGER NOT NULL );",
			Expect: "ALTER TABLE `fuga` ADD COLUMN `a` INT (11) NOT NULL AFTER `id`;\nALTER TABLE `fuga` ADD COLUMN `b` INT (11) NOT NULL AFTER `a`;\nALTER TABLE `fuga` ADD COLUMN `c` INT (11) NOT NULL AFTER `b`;",
		},
		// add column (first)
		{
			Before: "CREATE TABLE `fuga` ( `id` INTEGER NOT NULL );",
			After:  "CREATE TABLE `fuga` ( `a` INTEGER NOT NULL, `b` INTEGER NOT NULL, `c` INTEGER NOT NULL, `id` INTEGER NOT NULL);",
			Expect: "ALTER TABLE `fuga` ADD COLUMN `a` INT (11) NOT NULL FIRST;\nALTER TABLE `fuga` ADD COLUMN `b` INT (11) NOT NULL AFTER `a`;\nALTER TABLE `fuga` ADD COLUMN `c` INT (11) NOT NULL AFTER `b`;",
		},
		{
			Before: "CREATE TABLE `fuga` ( `id` INTEGER NOT NULL );",
			After:  "CREATE TABLE `fuga` ( `id` INTEGER NOT NULL, `c` INTEGER NOT NULL, `a` INTEGER NOT NULL, `b` INTEGER NOT NULL );",
			Expect: "ALTER TABLE `fuga` ADD COLUMN `c` INT (11) NOT NULL AFTER `id`;\nALTER TABLE `fuga` ADD COLUMN `a` INT (11) NOT NULL AFTER `c`;\nALTER TABLE `fuga` ADD COLUMN `b` INT (11) NOT NULL AFTER `a`;",
		},
		// change column
		{
			Before: "CREATE TABLE `fuga` ( `id` INTEGER NOT NULL );",
			After:  "CREATE TABLE `fuga` ( `id` BIGINT NOT NULL );",
			Expect: "ALTER TABLE `fuga` CHANGE COLUMN `id` `id` BIGINT (20) NOT NULL;",
		},
		// change column with comment
		{
			Before: "CREATE TABLE `fuga` ( `id` INTEGER NOT NULL );",
			After:  "CREATE TABLE `fuga` ( `id` INTEGER NOT NULL COMMENT 'fuga is good' );",
			Expect: "ALTER TABLE `fuga` CHANGE COLUMN `id` `id` INT (11) NOT NULL COMMENT 'fuga is good';",
		},
		// drop primary key
		{
			Before: "CREATE TABLE `fuga` ( `id` INTEGER NOT NULL AUTO_INCREMENT, PRIMARY KEY (`id`) );",
			After:  "CREATE TABLE `fuga` ( `id` INTEGER NOT NULL AUTO_INCREMENT );",
			Expect: "ALTER TABLE `fuga` DROP PRIMARY KEY;",
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
			Expect: "ALTER TABLE `fuga` DROP FOREIGN KEY `fsym`;\nALTER TABLE `fuga` DROP INDEX `fsym`;\nALTER TABLE `fuga` ADD INDEX `ksym` (`fid`);\nALTER TABLE `fuga` ADD CONSTRAINT `ksym` FOREIGN KEY (`fid`) REFERENCES `f` (`id`);",
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
			Expect: "ALTER TABLE `fuga` DROP INDEX `ab`;\nALTER TABLE `fuga` DROP COLUMN `bid`;\nALTER TABLE `fuga` ADD COLUMN `cid` INT (11) NOT NULL AFTER `aid`;\nALTER TABLE `fuga` ADD INDEX `ac` (`aid`, `cid`);",
		},
		// not change to query what generated by show create table
		{
			// human input
			Before: `
create table foo (
   id int not null AUTO_INCREMENT PRIMARY KEY,
   tinyints tinyint,
   tinyintu tinyint unsigned,
   smallints smallint,
   smallintu smallint unsigned,
   mediumints mediumint,
   mediumintu mediumint unsigned,
   ints int comment 'this is sined int nullable',
   intu int unsigned,
   integers integer null default null,
   integeru integer unsigned null,
   bigins bigint UNIQUE KEY,
   bigintu bigint unsigned,
   floats float,
   floaru float unsigned,
   doubles double,
   doubleu double unsigned,
   decimals decimal,
   decimalu decimal unsigned,
   varcharn varchar (10) null,
   varcharnn varchar (10) not null,
   textn text,
   textnn text not null,
   blobn blob,
   blobnn blob,
   intsd int default 0,
   intud int unsigned default 0,
   CONSTRAINT bar_fk FOREIGN KEY (integers) REFERENCES bar (id),
   INDEX foo_idx (ints)
);
			`,
			// show create table foo
			After: `
CREATE TABLE foo (
  id int(11) NOT NULL AUTO_INCREMENT,
  tinyints tinyint(4) DEFAULT NULL,
  tinyintu tinyint(3) unsigned DEFAULT NULL,
  smallints smallint(6) DEFAULT NULL,
  smallintu smallint(5) unsigned DEFAULT NULL,
  mediumints mediumint(9) DEFAULT NULL,
  mediumintu mediumint(8) unsigned DEFAULT NULL,
  ints int(11) DEFAULT NULL COMMENT 'this is sined int nullable',
  intu int(10) unsigned DEFAULT NULL,
  integers int(11) DEFAULT NULL,
  integeru int(10) unsigned DEFAULT NULL,
  bigins bigint(20) DEFAULT NULL,
  bigintu bigint(20) unsigned DEFAULT NULL,
  floats float DEFAULT NULL,
  floaru float unsigned DEFAULT NULL,
  doubles double DEFAULT NULL,
  doubleu double unsigned DEFAULT NULL,
  decimals decimal(10,0) DEFAULT NULL,
  decimalu decimal(10,0) unsigned DEFAULT NULL,
  varcharn varchar(10) DEFAULT NULL,
  varcharnn varchar(10) NOT NULL,
  textn text,
  textnn text NOT NULL,
  blobn blob,
  blobnn blob,
  intsd int(11) DEFAULT '0',
  intud int(10) unsigned DEFAULT '0',
  PRIMARY KEY (id),
  UNIQUE KEY bigins (bigins),
  KEY bar_fk (integers),
  KEY foo_idx (ints),
  CONSTRAINT bar_fk FOREIGN KEY (integers) REFERENCES bar (id)
);
			`,
			Expect: "",
		},
	}

	var buf bytes.Buffer
	for _, spec := range specs {
		buf.Reset()
		if !assert.NoError(t, diff.Strings(&buf, spec.Before, spec.After), "diff.String should succeed") {
			return
		}

		if !assert.Equal(t, spec.Expect, buf.String(), "result SQL should match") {
			t.Logf("before = %s", spec.Before)
			t.Logf("after = %s", spec.After)
			return
		}
	}
}
