package schemalex

import (
	"bytes"
	"io"

	"github.com/schemalex/schemalex/internal/util"
	"github.com/schemalex/schemalex/statement"
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

func (c *CreateTableStatement) ID() string {
	return c.Name
}

func (c *CreateTableStatement) LookupColumn(name string) (statement.TableColumn, bool) {
	for _, col := range c.Columns {
		if col.Name() == name {
			return col, true
		}
	}
	return nil, false
}

func (c *CreateTableStatement) LookupIndex(name string) (statement.Index, bool) {
	for _, idx := range c.Indexes {
		// TODO: This is wacky. fix how we match an index
		if idx.String() == name {
			return idx, true
		}
	}
	return nil, false
}

func (c *CreateTableStatement) WriteTo(dst io.Writer) (int64, error) {
	var b bytes.Buffer

	b.WriteString("CREATE")
	if c.Temporary {
		b.WriteString(" TEMPORARY")
	}

	b.WriteString(" TABLE")
	if c.IfNotExist {
		b.WriteString(" IF NOT EXISTS")
	}

	b.WriteByte(' ')
	b.WriteString(util.Backquote(c.Name))
	b.WriteString(" (")

	fields := make([]Stmt, 0, len(c.Columns)+len(c.Indexes))
	for _, col := range c.Columns {
		fields = append(fields, col)
	}
	for _, idx := range c.Indexes {
		fields = append(fields, idx)
	}

	for i, stmt := range fields {
		b.WriteByte('\n')
		if _, err := stmt.WriteTo(&b); err != nil {
			return 0, err
		}
		if i < len(fields)-1 {
			b.WriteByte(',')
		}
	}

	b.WriteString("\n)")

	if l := len(c.Options); l > 0 {
		b.WriteByte(' ')
		for i, option := range c.Options {
			if _, err := option.WriteTo(&b); err != nil {
				return 0, err
			}

			if i < l-1 {
				b.WriteString(", ")
			}
		}
	}

	return b.WriteTo(dst)
}

func (c *CreateTableOptionStatement) WriteTo(dst io.Writer) (int64, error) {
	var buf bytes.Buffer
	buf.WriteString(c.Key)
	buf.WriteString(" = ")
	buf.WriteString(c.Value)

	return buf.WriteTo(dst)
}
