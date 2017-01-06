package main

import (
	"flag"
	"log"
	"os"

	"github.com/lestrrat/schemalex"
	"github.com/lestrrat/schemalex/diff"
)

var (
	errorMarker  = flag.String("error-marker", "___", "marker of parse error position")
	errorContext = flag.Int("error-context", 20, "before, after context position of parse error")
)

func main() {
	flag.Parse()
	args := flag.Args()

	if len(args) != 2 {
		log.Fatalf("should call schemalex <options> /path/to/before /path/to/after")
	}

	if err := _main(args[0], args[1]); err != nil {
		log.Fatal(err)
	}
}

func _main(before, after string) error {
	p := schemalex.New()
	p.ErrorMarker = *errorMarker
	p.ErrorContext = *errorContext

	return diff.Files(os.Stderr, before, after, diff.WithParser(p))
}
