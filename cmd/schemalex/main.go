package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/lestrrat/schemalex"
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
	b, err := ioutil.ReadFile(before)
	if err != nil {
		return err
	}

	a, err := ioutil.ReadFile(after)
	if err != nil {
		return err
	}

	p := schemalex.NewParser(string(b))
	p.ErrorMarker = *errorMarker
	p.ErrorContext = *errorContext

	beforeStmts, err := p.Parse()
	if err != nil {
		return fmt.Errorf("file:%s error:%s", before, err)
	}

	p = schemalex.NewParser(string(a))
	p.ErrorMarker = *errorMarker
	p.ErrorContext = *errorContext

	afterStmts, err := p.Parse()
	if err != nil {
		return fmt.Errorf("file:%s error:%s", after, err)
	}

	d := &schemalex.Differ{filterCreateTableStatement(beforeStmts), filterCreateTableStatement(afterStmts)}
	return d.WriteDiffWithTransaction(os.Stdout)
}

func filterCreateTableStatement(stmts []schemalex.Stmt) []schemalex.CreateTableStatement {
	var createTableStatements []schemalex.CreateTableStatement
	for _, stmt := range stmts {
		switch t := stmt.(type) {
		case *schemalex.CreateTableStatement:
			createTableStatements = append(createTableStatements, *t)
		}
	}
	return createTableStatements
}
