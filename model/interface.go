package model

import "io"

type Stmt interface {
	io.WriterTo
	ID() string
}
type Stmts []Stmt

type maybeString struct {
	Valid bool
	Value string
}

type ColumnContainer interface {
	AddColumns(...string)
	Columns() chan string
}

type Index interface {
	Stmt
	ColumnContainer

	HasName() bool
	HasSymbol() bool
	Name() string
	String() string
	Reference() Reference
	SetReference(Reference)
	SetSymbol(string)
	SetType(IndexType)
	SetName(string)
	Symbol() string
	IsBtree() bool
	IsHash() bool
	IsPrimaryKey() bool
	IsNormal() bool
	IsUnique() bool
	IsFullText() bool
	IsSpatial() bool
	IsForeginKey() bool
}

type IndexKind int

const (
	IndexKindInvalid IndexKind = iota
	IndexKindPrimaryKey
	IndexKindNormal
	IndexKindUnique
	IndexKindFullText
	IndexKindSpatial
	IndexKindForeignKey
)

type IndexType int

const (
	IndexTypeNone IndexType = iota
	IndexTypeBtree
	IndexTypeHash
)

type index struct {
	symbol  maybeString
	kind    IndexKind
	name    maybeString
	typ     IndexType
	columns []string
	// TODO Options.
	reference Reference
}

type Reference interface {
	ColumnContainer

	String() string
	TableName() string
	OnDelete() ReferenceOption
	OnUpdate() ReferenceOption
	SetTableName(string)
	SetMatch(ReferenceMatch)
	SetOnDelete(ReferenceOption)
	SetOnUpdate(ReferenceOption)
	MatchFull() bool
	MatchPartial() bool
	MatchSimple() bool
}

type reference struct {
	tableName string
	columns   []string
	match     ReferenceMatch
	onDelete  ReferenceOption
	onUpdate  ReferenceOption
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

type Table interface {
	Stmt

	Name() string
	IsTemporary() bool
	SetTemporary(bool)
	IsIfNotExists() bool
	SetIfNotExists(bool)

	AddColumn(TableColumn)
	Columns() chan TableColumn
	AddIndex(Index)
	Indexes() chan Index
	AddOption(TableOption)
	Options() chan TableOption

	LookupColumn(string) (TableColumn, bool)
	LookupIndex(string) (Index, bool)
}

type TableOption interface {
	Stmt
	Key() string
	Value() string
}

type table struct {
	name        string
	temporary   bool
	ifnotexists bool
	columns     []TableColumn
	indexes     []Index
	options     []TableOption
}

type tableopt struct {
	key   string
	value string
}

type NullState int

const (
	NullStateNone NullState = iota
	NullStateNull
	NullStateNotNull
)

type Length interface {
	HasDecimal() bool
	Decimal() string
	SetDecimal(string)
	Length() string
}

type length struct {
	decimals maybeString
	length   string
}

type TableColumn interface {
	Stmt

	Name() string
	Type() ColumnType
	SetType(ColumnType)

	HasLength() bool
	Length() Length
	SetLength(Length)
	HasCharacterSet() bool
	CharacterSet() string
	HasCollation() bool
	Collation() string
	HasDefault() bool
	Default() string
	IsQuotedDefault() bool
	SetDefault(string, bool)
	HasComment() bool
	Comment() string
	SetComment(string)

	NullState() NullState
	SetNullState(NullState)

	IsAutoIncrement() bool
	SetAutoIncrement(bool)
	IsBinary() bool
	SetBinary(bool)
	IsKey() bool
	SetKey(bool)
	IsPrimary() bool
	SetPrimary(bool)
	IsUnique() bool
	SetUnique(bool)
	IsUnsigned() bool
	SetUnsigned(bool)
	IsZeroFill() bool
	SetZeroFill(bool)
}

type defaultValue struct {
	Valid  bool
	Value  string
	Quoted bool
}

type tablecol struct {
	name         string
	typ          ColumnType
	length       Length
	nullstate    NullState
	charset      maybeString
	collation    maybeString
	defaultValue defaultValue
	comment      maybeString
	autoincr     bool
	binary       bool
	key          bool
	primary      bool
	unique       bool
	unsigned     bool
	zerofill     bool
}

type Database interface {
	Stmt

	Name() string
	IsIfNotExists() bool
	SetIfNotExists(bool)
}

type database struct {
	name        string
	ifnotexists bool
}
