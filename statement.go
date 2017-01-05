package schemalex

import (
	"bytes"
	"fmt"
	"strings"
)

func (c *CreateDatabaseStatement) String() string {
	var buf bytes.Buffer

	buf.WriteString("CREATE DATABASE")
	if c.IfNotExist {
		buf.WriteString(" IF NOT EXISTS")
	}
	buf.WriteByte(' ')
	buf.WriteString(Backquote(c.Name))
	buf.WriteByte(';')
	return buf.String()
}

func (c *CreateTableStatement) String() string {
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
	b.WriteString(" (\n")

	var fields []string

	for _, columnStatement := range c.Columns {
		fields = append(fields, columnStatement.String())
	}

	for _, indexStatement := range c.Indexes {
		fields = append(fields, indexStatement.String())
	}

	b.WriteString(strings.Join(fields, ",\n"))

	b.WriteString("\n)")

	var options []string

	for _, optionStatement := range c.Options {
		options = append(options, optionStatement.String())
	}

	if str := strings.Join(options, ", "); str != "" {
		b.WriteString(" " + str)
	}

	return b.String()
}

func (c *CreateTableOptionStatement) String() string {
	return fmt.Sprintf("%s = %s", c.Key, c.Value)
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

func (c *CreateTableColumnStatement) String() string {
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

	return buf.String()
}

func (l *Length) String() string {
	if l.Decimals.Valid {
		return fmt.Sprintf("%s, %s", l.Length, l.Decimals)
	}
	return l.Length
}

func (c *CreateTableIndexStatement) String() string {
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
		if i < len(c.ColNames) - 1 {
			buf.WriteString(", ")
		}
	}
	buf.WriteByte(')')

	if c.Reference != nil {
		buf.WriteByte(' ')
		buf.WriteString(c.Reference.String())
	}

	return buf.String()
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
		if i < len(r.ColNames) - 1 {
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
