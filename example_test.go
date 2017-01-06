package schemalex_test

import (
	"os"

	"github.com/schemalex/schemalex/diff"
)

func Example() {
	const sql1 = `CREATE TABLE hoge (
    id INTEGER NOT NULL AUTO_INCREMENT,
    PRIMARY KEY (id)
);`
	const sql2 = `CREATE TABLE hoge (
    id INTEGER NOT NULL AUTO_INCREMENT,
    c VARCHAR (20) NOT NULL DEFAULT "hoge",
    PRIMARY KEY (id)
);

CREATE TABLE fuga (
    id INTEGER NOT NULL AUTO_INCREMENT,
    PRIMARY KEY (id)
);`

	diff.Strings(os.Stdout, sql1, sql2, diff.WithTransaction(true))

	// OUTPUT:
	// BEGIN;
	//
	// SET FOREIGN_KEY_CHECKS = 0;
	//
	// CREATE TABLE `fuga` (
	// `id` INTEGER NOT NULL AUTO_INCREMENT,
	// PRIMARY KEY (`id`)
	// );
	//
	// ALTER TABLE `hoge` ADD COLUMN `c` VARCHAR (20) NOT NULL DEFAULT "hoge";
	//
	// SET FOREIGN_KEY_CHECKS = 1;
	//
	// COMMIT;
}
