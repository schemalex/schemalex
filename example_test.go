package schemalex_test

import (
	"os"

	"github.com/lestrrat/schemalex"
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

	s := schemalex.New()
	stmts1, _ := s.ParseString(sql1)
	stmts2, _ := s.ParseString(sql2)
	schemalex.Diff(os.Stdout, stmts1, stmts2)

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