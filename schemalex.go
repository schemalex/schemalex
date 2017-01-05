//go:generate go run internal/cmd/gentokens/main.go

package schemalex

import "io"

// Backquote surrounds the given string in backquotes
func Backquote(s string) string {
	// XXX Does this require escaping
	return "`" + s + "`"
}


func filterCreateTableStatement(stmts []Stmt) []CreateTableStatement {
	var createTableStatements []CreateTableStatement
	for _, stmt := range stmts {
		switch t := stmt.(type) {
		case *CreateTableStatement:
			createTableStatements = append(createTableStatements, *t)
		}
	}
	return createTableStatements
}

func Diff(dst io.Writer, from, to []Stmt) error {
	d := Differ{
		filterCreateTableStatement(from),
		filterCreateTableStatement(to),
	}
	return d.WriteDiffWithTransaction(dst)
}
