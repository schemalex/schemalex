package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"

	"github.com/schemalex/schemalex"
	"github.com/schemalex/schemalex/diff"
	"github.com/schemalex/schemalex/internal/errors"
)

func main() {
	if err := _main(); err != nil {
		log.Fatal(err)
	}
}

func _main() error {
	var txn bool
	var version bool
	var outfile string

	flag.Usage = func() {
		fmt.Printf(`schemalex version %s

schemalex -version
schemalex [options...] /path/to/before /path/to/after

-v            Print out the version and exit
-o file	      Output the result to the specified file (default: stdout)
-t[=true]     Enable/Disable transaction in the output (default: true)
`, schemalex.Version)
	}
	flag.BoolVar(&version, "v", false, "")
	flag.BoolVar(&txn, "t", true, "")
	flag.StringVar(&outfile, "o", "", "")
	flag.Parse()

	if version {
		fmt.Printf(
			"schemalex version %s, built with go %s for %s/%s\n",
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

	p := schemalex.New()
	return diff.Files(
		dst,
		flag.Arg(0),
		flag.Arg(1),
		diff.WithTransaction(txn), diff.WithParser(p),
	)
}
