# schemalex

Generate sql of difference two mysql schema

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
$ schemalex --before /path/to/sql --after /path/to/sql
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
    * FOREIGN KEY
    * CHECK (expr)
* Type
    * SET
    * ENUM
    * SPATIAL type
* INDEX OPTION
* DATABASE CHARACTER
* REFACTORING

## SEE ALSO

* http://rspace.googlecode.com/hg/slide/lex.html#landing-slide
* http://blog.gopheracademy.com/advent-2014/parsers-lexers/
