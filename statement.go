package schemalex

import (
	"bytes"
	"fmt"
	"io"
)

type ider interface {
	ID() string
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
	buf.WriteString(Backquote(c.Name))
	buf.WriteByte(';')

	return buf.WriteTo(dst)
}

func (c *CreateTableStatement) ID() string {
	return c.Name
}

func (c *CreateTableStatement) LookupColumn(name string) (*CreateTableColumnStatement, bool) {
	for _, col := range c.Columns {
		if col.Name == name {
			return col, true
		}
	}
	return nil, false
}

func (c *CreateTableStatement) LookupIndex(name string) (*CreateTableIndexStatement, bool) {
	for _, idx := range c.Indexes {
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
	b.WriteString(Backquote(c.Name))
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

func (c ColumnOptionNullState) String() string {
	switch c {
	case ColumnOptionNullStateNone:
		return ""
	case ColumnOptionNullStateNull:
		return "NULL"
	case ColumnOptionNullStateNotNull:
		return "NOT NULL"
	default:
		panic("not reach")
	}
}

func (c CreateTableColumnStatement) WriteTo(dst io.Writer) (int64, error) {
	var buf bytes.Buffer

	buf.WriteString(Backquote(c.Name))
	buf.WriteByte(' ')
	buf.WriteString(c.Type.String())

	if c.Length.Valid {
		buf.WriteString(" (")
		buf.WriteString(c.Length.String())
		buf.WriteByte(')')
	}

	if c.Unsgined {
		buf.WriteString(" UNSIGNED")
	}

	if c.ZeroFill {
		buf.WriteString(" ZEROFILL")
	}

	if c.Binary {
		buf.WriteString(" BINARY")
	}

	if c.CharacterSet.Valid {
		buf.WriteString(" CHARACTER SET ")
		buf.WriteString(Backquote(c.CharacterSet.Value))
	}

	if c.Collate.Valid {
		buf.WriteString(" COLLATE ")
		buf.WriteString(Backquote(c.Collate.Value))
	}

	if str := c.Null.String(); str != "" {
		buf.WriteByte(' ')
		buf.WriteString(str)
	}

	if c.Default.Valid {
		buf.WriteString(" DEFAULT ")
		buf.WriteString(c.Default.Value)
	}

	if c.AutoIncrement {
		buf.WriteString(" AUTO_INCREMENT")
	}

	if c.Unique {
		buf.WriteString(" UNIQUE KEY")
	}

	if c.Primary {
		buf.WriteString(" PRIMARY KEY")
	}

	if c.Key {
		buf.WriteString(" KEY")
	}

	if c.Comment.Valid {
		buf.WriteString(" '")
		buf.WriteString(c.Comment.Value)
		buf.WriteByte('\'')
	}

	return buf.WriteTo(dst)
}

func (l *Length) String() string {
	if l.Decimals.Valid {
		return fmt.Sprintf("%s, %s", l.Length, l.Decimals)
	}
	return l.Length
}

func (c CreateTableIndexStatement) String() string {
	var buf bytes.Buffer
	c.WriteTo(&buf)
	return buf.String()
}

func (c CreateTableIndexStatement) WriteTo(dst io.Writer) (int64, error) {
	var buf bytes.Buffer

	if c.Symbol.Valid {
		buf.WriteString("CONSTRAINT ")
		buf.WriteString(Backquote(c.Symbol.Value))
		buf.WriteByte(' ')
	}

	buf.WriteString(c.Kind.String())

	if c.Name.Valid {
		buf.WriteByte(' ')
		buf.WriteString(Backquote(c.Name.Value))
	}

	if str := c.Type.String(); str != "" {
		buf.WriteByte(' ')
		buf.WriteString(str)
	}

	buf.WriteString(" (")
	for i, colName := range c.ColNames {
		buf.WriteString(Backquote(colName))
		if i < len(c.ColNames)-1 {
			buf.WriteString(", ")
		}
	}
	buf.WriteByte(')')

	if c.Reference != nil {
		buf.WriteByte(' ')
		buf.WriteString(c.Reference.String())
	}

	return buf.WriteTo(dst)
}

func (i IndexKind) String() string {
	switch i {
	case IndexKindPrimaryKey:
		return "PRIMARY KEY"
	case IndexKindNormal:
		return "INDEX"
	case IndexKindUnique:
		return "UNIQUE INDEX"
	case IndexKindFullText:
		return "FULLTEXT INDEX"
	case IndexKindSpartial:
		return "SPARTIAL INDEX"
	case IndexKindForeignKey:
		return "FOREIGN KEY"
	default:
		panic("not reach")
	}
}

func (i IndexType) String() string {
	switch i {
	case IndexTypeNone:
		return ""
	case IndexTypeBtree:
		return "USING BTREE"
	case IndexTypeHash:
		return "USING HASH"
	default:
		panic("not reach")
	}
}

func (r *Reference) String() string {
	var buf bytes.Buffer

	buf.WriteString("REFERENCES ")
	buf.WriteString(Backquote(r.TableName))
	buf.WriteString(" (")
	for i, colName := range r.ColNames {
		buf.WriteString(Backquote(colName))
		if i < len(r.ColNames)-1 {
			buf.WriteString(", ")
		}
	}
	buf.WriteByte(')')

	if str := r.Match.String(); str != "" {
		buf.WriteByte(' ')
		buf.WriteString(str)
	}

	if r.OnDelete != ReferenceOptionNone {
		buf.WriteString(" ON DELETE ")
		buf.WriteString(r.OnDelete.String())
	}

	if r.OnUpdate != ReferenceOptionNone {
		buf.WriteString(" ON UPDATE ")
		buf.WriteString(r.OnUpdate.String())
	}

	return buf.String()
}

func (r ReferenceMatch) String() string {
	switch r {
	case ReferenceMatchNone:
		return ""
	case ReferenceMatchFull:
		return "MATCH FULL"
	case ReferenceMatchPartial:
		return "MATCH PARTIAL"
	case ReferenceMatchSimple:
		return "MATCH SIMPLE"
	default:
		panic("not reach")
	}
}

func (r ReferenceOption) String() string {
	switch r {
	case ReferenceOptionNone:
		return ""
	case ReferenceOptionRestrict:
		return "RESTRICT"
	case ReferenceOptionCascade:
		return "CASCADE"
	case ReferenceOptionSetNull:
		return "SET NULL"
	case ReferenceOptionNoAction:
		return "NO ACTION"
	default:
		panic("not reach")
	}
}
