package schemalex

import (
	"bytes"
	"fmt"
	"strings"
)

type Stmt interface {
	String() string
}

type CreateDatabaseStatement struct {
	Name       string
	IfNotExist bool
}

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

type CreateTableStatement struct {
	Name       string
	Temporary  bool
	IfNotExist bool
	Columns    []CreateTableColumnStatement
	Indexes    []CreateTableIndexStatement
	Options    []CreateTableOptionStatement
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

type CreateTableOptionStatement struct {
	Key   string
	Value string
}

func (c *CreateTableOptionStatement) String() string {
	return fmt.Sprintf("%s = %s", c.Key, c.Value)
}

type ColumnOptionNullState int

const (
	ColumnOptionNullStateNone ColumnOptionNullState = iota
	ColumnOptionNullStateNull
	ColumnOptionNullStateNotNull
)

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

type MaybeString struct {
	Valid bool
	Value string
}

type CreateTableColumnStatement struct {
	Name          string
	Type          ColumnType
	Length        Length
	Unsgined      bool
	ZeroFill      bool
	Binary        bool
	CharacterSet  MaybeString
	Collate       MaybeString
	Null          ColumnOptionNullState
	Default       MaybeString
	AutoIncrement bool
	Unique        bool
	Primary       bool
	Key           bool
	Comment       MaybeString
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

const (
	ColumnOptionSize = 1 << iota
	ColumnOptionDecimalSize
	ColumnOptionDecimalOptionalSize
	ColumnOptionUnsigned
	ColumnOptionZerofill
	ColumnOptionBinary
	ColumnOptionCharacterSet
	ColumnOptionCollate
	ColumnOptionNull
	ColumnOptionDefault
	ColumnOptionAutoIncrement
	ColumnOptionKey
	ColumnOptionComment
)

const (
	ColumnOptionFlagNone            = 0
	ColumnOptionFlagDigit           = ColumnOptionSize | ColumnOptionUnsigned | ColumnOptionZerofill
	ColumnOptionFlagDecimal         = ColumnOptionDecimalSize | ColumnOptionUnsigned | ColumnOptionZerofill
	ColumnOptionFlagDecimalOptional = ColumnOptionDecimalOptionalSize | ColumnOptionUnsigned | ColumnOptionZerofill
	ColumnOptionFlagTime            = ColumnOptionSize
	ColumnOptionFlagChar            = ColumnOptionSize | ColumnOptionBinary | ColumnOptionCharacterSet | ColumnOptionCollate
	ColumnOptionFlagBinary          = ColumnOptionSize
)

type Length struct {
	Decimals MaybeString
	Length   string
	Valid    bool
}

func (l *Length) String() string {
	if l.Decimals.Valid {
		return fmt.Sprintf("%s, %s", l.Length, l.Decimals)
	}
	return l.Length
}

type CreateTableIndexStatement struct {
	Symbol   *string
	Kind     IndexKind
	Name     *string
	Type     IndexType
	ColNames []string
	// TODO Options.
	Reference *Reference
}

func (c *CreateTableIndexStatement) String() string {
	var strs []string

	if c.Symbol != nil {
		strs = append(strs, fmt.Sprintf("CONSTRAINT `%s`", *c.Symbol))
	}

	strs = append(strs, c.Kind.String())

	if c.Name != nil {
		strs = append(strs, fmt.Sprintf("`%s`", *c.Name))
	}

	if str := c.Type.String(); str != "" {
		strs = append(strs, str)
	}

	var cols []string

	for _, colName := range c.ColNames {
		cols = append(cols, fmt.Sprintf("`%s`", colName))
	}

	strs = append(strs, "("+strings.Join(cols, ", ")+")")

	if c.Reference != nil {
		strs = append(strs, c.Reference.String())
	}

	return strings.Join(strs, " ")
}

type IndexKind int

const (
	IndexKindPrimaryKey IndexKind = iota
	IndexKindNormal
	IndexKindUnique
	IndexKindFullText
	IndexKindSpartial
	IndexKindForeignKey
)

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

type IndexType int

const (
	IndexTypeNone IndexType = iota
	IndexTypeBtree
	IndexTypeHash
)

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

type Reference struct {
	TableName string
	ColNames  []string
	Match     ReferenceMatch
	OnDelete  ReferenceOption
	OnUpdate  ReferenceOption
}

func (r *Reference) String() string {
	var strs []string

	strs = append(strs, "REFERENCES")
	strs = append(strs, fmt.Sprintf("`%s`", r.TableName))

	var cols []string

	for _, colName := range r.ColNames {
		cols = append(cols, fmt.Sprintf("`%s`", colName))
	}

	strs = append(strs, "("+strings.Join(cols, ", ")+")")

	if str := r.Match.String(); str != "" {
		strs = append(strs, str)
	}

	if r.OnDelete != ReferenceOptionNone {
		strs = append(strs, fmt.Sprintf("ON DELETE %s", r.OnDelete.String()))
	}

	if r.OnUpdate != ReferenceOptionNone {
		strs = append(strs, fmt.Sprintf("ON UPDATE %s", r.OnUpdate.String()))
	}

	return strings.Join(strs, " ")
}

type ReferenceMatch int

const (
	ReferenceMatchNone ReferenceMatch = iota
	ReferenceMatchFull
	ReferenceMatchPartial
	ReferenceMatchSimple
)

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

type ReferenceOption int

const (
	ReferenceOptionNone ReferenceOption = iota
	ReferenceOptionRestrict
	ReferenceOptionCascade
	ReferenceOptionSetNull
	ReferenceOptionNoAction
)

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
