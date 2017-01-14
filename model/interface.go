package model

// Stmt is the interface to define a statement
type Stmt interface {
	ID() string
}

// Stmt describes a list of statements
type Stmts []Stmt

type maybeString struct {
	Valid bool
	Value string
}

// ColumnContainer is the interface for objects that can contain
// column names
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

// Database represents a database definition
type Database interface {
	// This is a dummy method to differentiate between Table/Database interfaces.
	// without this, the Database interface is a subset of Table interface,
	// and then you need to be aware to check for v.(model.Table) BEFORE
	// making a check for v.(model.Database), which is silly.
	// Once you include a dummy method like this that differs from the
	// other interface, Go happily thinks that the two are separate entities
	isDatabase() bool

	Stmt

	Name() string
	IsIfNotExists() bool
	SetIfNotExists(bool)
}

type database struct {
	name        string
	ifnotexists bool
}
