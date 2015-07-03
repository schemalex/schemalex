[![wercker status](https://app.wercker.com/status/13480237267a3517d152deb7fc7b6b2e/s/master "wercker status")](https://app.wercker.com/project/bykey/13480237267a3517d152deb7fc7b6b2e)

# schemalex

Generate difference sql of two mysql schema

## DESCRIPTION

It is private study project for me and still alpha quority. API will be changed.

### DOWNLOAD

```
$ go get github.com/soh335/schemalex/cmd/schemalex
```

### USAGE

before
```sql
CREATE TABLE `hoge` (
    `id` INTEGER NOT NULL AUTO_INCREMENT,
    PRIMARY KEY (`id`)
);
```

after
```sql
CREATE TABLE `hoge` (
    `id` INTEGER NOT NULL AUTO_INCREMENT,
    `c` VARCHAR (20) NOT NULL DEFAULT "hoge",
    PRIMARY KEY (`id`)
);

CREATE TABLE `fuga` (
    `id` INTEGER NOT NULL AUTO_INCREMENT,
    PRIMARY KEY (`id`)
);
```

```
$ schemalex <options> /path/to/before.sql /path/to/after.sql
```

output
```sql
BEGIN;

SET FOREIGN_KEY_CHECKS = 0;

CREATE TABLE `fuga` (
`id` INTEGER NOT NULL AUTO_INCREMENT,
PRIMARY KEY (`id`)
);

ALTER TABLE `hoge` ADD COLUMN `c` VARCHAR (20) NOT NULL DEFAULT "hoge";

SET FOREIGN_KEY_CHECKS = 1;

COMMIT;
```

## TODO

* COLUMN
    * CHECK (expr)
* Type
    * SET
    * ENUM
    * SPATIAL type
* INDEX OPTION
* DATABASE CHARACTER
* MANY, MANY REFACTORING

## SEE ALSO

* http://rspace.googlecode.com/hg/slide/lex.html#landing-slide
* http://blog.gopheracademy.com/advent-2014/parsers-lexers/
* https://github.com/soh335/git-schemalex

## LICENSE

MIT
