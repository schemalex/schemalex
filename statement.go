package schemalex

import (
	"bytes"
	"io"

	"github.com/schemalex/schemalex/internal/util"
)

type ider interface {
	ID() string
}

func (s Statements) WriteTo(dst io.Writer) (int64, error) {
	var n int64
	for _, stmt := range s {
		n1, err := stmt.WriteTo(dst)
		n += n1
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

// Lookup looks up statements by their ID, which could be their
// "name" or their stringified representation
func (s Statements) Lookup(id string) (Stmt, bool) {
	for _, stmt := range s {
		if n, ok := stmt.(ider); ok {
			if n.ID() == id {
				return stmt, true
			}
		}
	}
	return nil, false
}

func (c *CreateDatabaseStatement) ID() string {
	return c.Name
}

func (c *CreateDatabaseStatement) WriteTo(dst io.Writer) (int64, error) {
	var buf bytes.Buffer
	buf.WriteString("CREATE DATABASE")
	if c.IfNotExist {
		buf.WriteString(" IF NOT EXISTS")
	}
	buf.WriteByte(' ')
	buf.WriteString(util.Backquote(c.Name))
	buf.WriteByte(';')

	return buf.WriteTo(dst)
}
