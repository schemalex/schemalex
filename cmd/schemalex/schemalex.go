package main

import (
	"log"
	"os"

	"github.com/schemalex/schemalex"
	"github.com/schemalex/schemalex/diff"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatalf("should call schemalex <options> /path/to/before /path/to/after")
	}

	if err := _main(os.Args[1], os.Args[2]); err != nil {
		log.Fatal(err)
	}
}

func _main(before, after string) error {
	p := schemalex.New()
	return diff.Files(os.Stderr, before, after, diff.WithTransaction(true), diff.WithParser(p))
}
