package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"

	"github.com/pkg/errors"
	"github.com/eihigh/schemalex"
	"github.com/eihigh/schemalex/diff"
)

func main() {
	if err := _main(); err != nil {
		log.Printf("%s", err)
		os.Exit(1)
	}
}

func _main() error {
	var txn bool
	var version bool
	var outfile string

	flag.Usage = func() {
		fmt.Printf(`schemadiff version %s

schemadiff -version
schemadiff [options...] before after

-v            Print out the version and exit
-o file	      Output the result to the specified file (default: stdout)
-t[=true]     Enable/Disable transaction in the output (default: true)

"before" and "after" may be a file path, or a URI.
Special URI schemes "mysql" and "local-git" are supported on top of
"file". If the special path "-" is used, it is treated as stdin

Examples:

* Compare local files
  schemadiff /path/to/file /another/path/to/file
  schemadiff file:///path/to/file /another/path/to/file

* Compare local file against online mysql schema
  schemadiff /path/to/file "mysql://user:password@tcp(host:port)/dbname?option=value"

* Compare file in local git repository against local file
  schemadiff "local-git:///path/to/repo?file=foo.sql&commitish=deadbeaf" /path/to/file

* Compare schema from stdin against local file
	.... | schemadiff - /path/to/file

`, schemalex.Version)
	}
	flag.BoolVar(&version, "v", false, "")
	flag.BoolVar(&txn, "t", true, "")
	flag.StringVar(&outfile, "o", "", "")
	flag.Parse()

	if version {
		fmt.Printf(
			"schemadiff version %s, built with go %s for %s/%s\n",
			schemalex.Version,
			runtime.Version(),
			runtime.GOOS,
			runtime.GOARCH,
		)
		return nil
	}

	if flag.NArg() != 2 {
		flag.Usage()
		return errors.New("wrong number of arguments")
	}

	var dst io.Writer = os.Stdout
	if len(outfile) > 0 {
		f, err := os.OpenFile(outfile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return errors.Wrapf(err, `failed to open file %s for writing`, outfile)
		}
		dst = f
		defer f.Close()
	}

	fromSource, err := schemalex.NewSchemaSource(flag.Arg(0))
	if err != nil {
		return errors.Wrap(err, `failed to create schema source for "from"`)
	}

	toSource, err := schemalex.NewSchemaSource(flag.Arg(1))
	if err != nil {
		return errors.Wrap(err, `failed to create schema source for "to"`)
	}

	p := schemalex.New()
	return diff.Sources(
		dst,
		fromSource,
		toSource,
		diff.WithTransaction(txn), diff.WithParser(p),
	)
}
