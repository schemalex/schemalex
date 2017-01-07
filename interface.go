package schemalex

import "io"

type Stmt interface {
	WriteTo(io.Writer) (int64, error)
}

type Statements []Stmt

type CreateDatabaseStatement struct {
	Name       string
	IfNotExist bool
}

type CreateTableStatement struct {
	Name       string
	Temporary  bool
	IfNotExist bool
	Columns    []*CreateTableColumnStatement
	Indexes    []*CreateTableIndexStatement
	Options    []*CreateTableOptionStatement
}

type CreateTableOptionStatement struct {
	Key   string
	Value string
}


// XXX need a comment
type coloptNullState int

const (
	coloptNullStateNone coloptNullState = iota
	coloptNullStateNull
	coloptNullStateNotNull
)

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
	Null          coloptNullState
	Default       MaybeString
	AutoIncrement bool
	Unique        bool
	Primary       bool
	Key           bool
	Comment       MaybeString
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

type CreateTableIndexStatement struct {
	Symbol   MaybeString
	Kind     IndexKind
	Name     MaybeString
	Type     IndexType
	ColNames []string
	// TODO Options.
	Reference *Reference
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

type IndexType int

const (
	IndexTypeNone IndexType = iota
	IndexTypeBtree
	IndexTypeHash
)

type Reference struct {
	TableName string
	ColNames  []string
	Match     ReferenceMatch
	OnDelete  ReferenceOption
	OnUpdate  ReferenceOption
}

type ReferenceMatch int

const (
	ReferenceMatchNone ReferenceMatch = iota
	ReferenceMatchFull
	ReferenceMatchPartial
	ReferenceMatchSimple
)

type ReferenceOption int

const (
	ReferenceOptionNone ReferenceOption = iota
	ReferenceOptionRestrict
	ReferenceOptionCascade
	ReferenceOptionSetNull
	ReferenceOptionNoAction
)
