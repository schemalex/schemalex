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
	var strs []string

	strs = append(strs, "CREATE DATABASE")

	if c.IfNotExist {
		strs = append(strs, "IF NOT EXISTS")
	}

	strs = append(strs, fmt.Sprintf("`%s`", c.Name))

	return strings.Join(strs, " ") + ";"
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

	b.WriteString(fmt.Sprintf(" `%s`", c.Name))
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

type CreateTableColumnStatement struct {
	Name          string
	Type          ColumnType
	Length        Length
	Unsgined      bool
	ZeroFill      bool
	Binary        bool
	CharacterSet  *string
	Collate       *string
	Null          ColumnOptionNullState
	Default       *string
	AutoIncrement bool
	Unique        bool
	Primary       bool
	Key           bool
	Comment       *string
}

func (c *CreateTableColumnStatement) String() string {
	var strs []string

	strs = append(strs, fmt.Sprintf("`%s`", c.Name))
	strs = append(strs, c.Type.String())

	if c.Length != nil {
		strs = append(strs, fmt.Sprintf("(%s)", c.Length.String()))
	}

	if c.Unsgined {
		strs = append(strs, "UNSIGNED")
	}

	if c.ZeroFill {
		strs = append(strs, "ZEROFILL")
	}

	if c.Binary {
		strs = append(strs, "BINARY")
	}

	if c.CharacterSet != nil {
		strs = append(strs, fmt.Sprintf("CHARACTER SET `%s`", *c.CharacterSet))
	}

	if c.Collate != nil {
		strs = append(strs, fmt.Sprintf("COLLATE `%s`", *c.Collate))
	}

	if str := c.Null.String(); str != "" {
		strs = append(strs, str)
	}

	if c.Default != nil {
		strs = append(strs, fmt.Sprintf("DEFAULT %s", *c.Default))
	}

	if c.AutoIncrement {
		strs = append(strs, "AUTO_INCREMENT")
	}

	if c.Unique {
		strs = append(strs, "UNIQUE KEY")
	}

	if c.Primary {
		strs = append(strs, "PRIMARY KEY")
	}

	if c.Key {
		strs = append(strs, "KEY")
	}

	if c.Comment != nil {
		strs = append(strs, fmt.Sprintf("'%s'", *c.Comment))
	}

	return strings.Join(strs, " ")
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

type Length interface {
	String() string
}

type LengthNumber struct{ Length string }

func (l *LengthNumber) String() string {
	return l.Length
}

type LengthDecimal struct {
	Length   string
	Decimals string
}

func (l *LengthDecimal) String() string {
	return fmt.Sprintf("%s, %s", l.Length, l.Decimals)
}

type LengthOptionalDecimal struct {
	Length   string
	Decimals *string
}

func (l LengthOptionalDecimal) String() string {
	if l.Decimals == nil {
		return l.Length
	} else {
		return fmt.Sprintf("%s, %s", l.Length, *l.Decimals)
	}
}

type ColumnType int

const (
	ColumnTypeBit ColumnType = iota
	ColumnTypeTinyInt
	ColumnTypeSmallInt
	ColumnTypeMediumInt
	ColumnTypeInt
	ColumnTypeInteger
	ColumnTypeBigInt
	ColumnTypeReal
	ColumnTypeDouble
	ColumnTypeFloat
	ColumnTypeDecimal
	ColumnTypeNumeric
	ColumnTypeDate
	ColumnTypeTime
	ColumnTypeTimestamp
	ColumnTypeDateTime
	ColumnTypeYear
	ColumnTypeChar
	ColumnTypeVarChar
	ColumnTypeBinary
	ColumnTypeVarBinary
	ColumnTypeTinyBlob
	ColumnTypeBlob
	ColumnTypeMediumBlob
	ColumnTypeLongBlob
	ColumnTypeTinyText
	ColumnTypeText
	ColumnTypeMediumText
	ColumnTypeLongText
)

func (c ColumnType) String() string {
	switch c {
	case ColumnTypeBit:
		return "BIT"
	case ColumnTypeTinyInt:
		return "TINYINT"
	case ColumnTypeSmallInt:
		return "SMALLINT"
	case ColumnTypeMediumInt:
		return "MEDIUMINT"
	case ColumnTypeInt:
		return "INT"
	case ColumnTypeInteger:
		return "INTEGER"
	case ColumnTypeBigInt:
		return "BIGINT"
	case ColumnTypeReal:
		return "REAL"
	case ColumnTypeDouble:
		return "DOUBLE"
	case ColumnTypeFloat:
		return "FLOAT"
	case ColumnTypeDecimal:
		return "DECIMAL"
	case ColumnTypeNumeric:
		return "NUMERIC"
	case ColumnTypeDate:
		return "DATE"
	case ColumnTypeTime:
		return "TIME"
	case ColumnTypeTimestamp:
		return "TIMESTAMP"
	case ColumnTypeDateTime:
		return "DATETIME"
	case ColumnTypeYear:
		return "YEAR"
	case ColumnTypeChar:
		return "CHAR"
	case ColumnTypeVarChar:
		return "VARCHAR"
	case ColumnTypeBinary:
		return "BINARY"
	case ColumnTypeVarBinary:
		return "VARBINARY"
	case ColumnTypeTinyBlob:
		return "TINYBLOB"
	case ColumnTypeBlob:
		return "BLOB"
	case ColumnTypeMediumBlob:
		return "MEDIUMBLOB"
	case ColumnTypeLongBlob:
		return "LONGBLOB"
	case ColumnTypeTinyText:
		return "TINYTEXT"
	case ColumnTypeText:
		return "TEXT"
	case ColumnTypeMediumText:
		return "MEDIUMTEXT"
	case ColumnTypeLongText:
		return "LONGTEXT"
	default:
		panic("not reach")
	}
}

type CreateTableIndexStatement struct {
	Symbol   *string
	Kind     IndexKind
	Name     *string
	Type     IndexType
	ColNames []string
	// TODO Options.
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
