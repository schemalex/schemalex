# schemalex

Generate the difference of two mysql schema

[![Build Status](https://travis-ci.org/schemalex/schemalex.png?branch=master)](https://travis-ci.org/schemalex/schemalex)

[![GoDoc](https://godoc.org/github.com/schemalex/schemalex?status.svg)](https://godoc.org/github.com/schemalex/schemalex)

## SYNOPSIS

This tool can be used to generate the difference, or more precisely,
the statements required to migrate from/to, between two MySQL
schema.

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

You can also use URI formatted strings as the sources to compare,
which allow you to compare local files against online schema,
a version committed to your git repository against another version,
etc.

Please see the help command for a list

```
schemalex -version
schemalex [options...] before after

-v            Print out the version and exit
-o file	      Output the result to the specified file (default: stdout)
-t[=true]     Enable/Disable transaction in the output (default: true)

"before" and "after" may be a file path, or a URI.
Special URI schemes "mysql" and "local-git" are supported on top of
"file". If the special path "-" is used, it is treated as stdin

Examples:

* Compare local files
  schemalex /path/to/file /another/path/to/file
  schemalex file:///path/to/file /another/path/to/file

* Compare local file against online mysql schema
  schemalex /path/to/file "mysql://user:password@tcp(host:port)/dbname?option=value"

* Compare file in local git repository against local file
  schemalex "local-git:///path/to/repo?file=foo.sql&commitish=deadbeaf" /path/to/file

* Compare schema from stdin against local file
	.... | schemalex - /path/to/file
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
