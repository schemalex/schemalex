# schemalex

Generate difference sql of two mysql schema

[![Build Status](https://travis-ci.org/schemalex/schemalex.png?branch=master)](https://travis-ci.org/schemalex/schemalex)

[![GoDoc](https://godoc.org/github.com/schemalex/schemalex?status.svg)](https://godoc.org/github.com/schemalex/schemalex)

## SYNOPSIS (COMMAND LINE)

Suppose you have an existing SQL schema like the following:

```sql
CREATE TABLE hoge (
    id INTEGER NOT NULL AUTO_INCREMENT,
    PRIMARY KEY (id)
);
```

And you want "upgrade" your schema to the following:


```sql
CREATE TABLE hoge (
    id INTEGER NOT NULL AUTO_INCREMENT,
    c VARCHAR (20) NOT NULL DEFAULT "hoge",
    PRIMARY KEY (id)
);

CREATE TABLE fuga (
    id INTEGER NOT NULL AUTO_INCREMENT,
    PRIMARY KEY (id)
);
```

Using `schemalex` you can generate a set of commands to perform the migration:

```
schemalex old.sql new.sql

SET FOREIGN_KEY_CHECKS = 0;

CREATE TABLE `fuga` (
`id` INTEGER NOT NULL AUTO_INCREMENT,
PRIMARY KEY (`id`)
);

ALTER TABLE `hoge` ADD COLUMN `c` VARCHAR (20) NOT NULL DEFAULT "hoge";

SET FOREIGN_KEY_CHECKS = 1;

COMMIT;
```

## SYNOPSIS (Using the library)

Below is the equivalent of the previous SYNOPSIS.

```
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
```

## SEE ALSO

* http://rspace.googlecode.com/hg/slide/lex.html#landing-slide
* http://blog.gopheracademy.com/advent-2014/parsers-lexers/
* https://github.com/soh335/git-schemalex

## LICENSE

MIT
