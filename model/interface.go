//go:generate stringer -type=IndexType -output=index_type_string_gen.go
//go:generate stringer -type=IndexKind -output=index_kind_string_gen.go
//go:generate stringer -type=ReferenceMatch -output=reference_match_string_gen.go
//go:generate stringer -type=ReferenceOption -output=reference_option_string_gen.go

package model

import "sync"

// Stmt is the interface to define a statement
type Stmt interface {
	ID() string
}

// Stmts describes a list of statements
type Stmts []Stmt

type maybeString struct {
	Valid bool
	Value string
}

// ColumnContainer is the interface for objects that can contain
// column names
type ColumnContainer interface {
	AddColumns(...IndexColumn)
	Columns() chan IndexColumn
}

type IndexColumnSortDirection int

const (
	SortDirectionNone IndexColumnSortDirection = iota
	SortDirectionAscending
	SortDirectionDescending
)

// IndexColumn is a column name/length specification used in indexes
type IndexColumn interface {
	ID() string
	Name() string
	SetLength(string) IndexColumn
	HasLength() bool
	Length() string
	SetSortDirection(IndexColumnSortDirection)
	HasSortDirection() bool
	IsAscending() bool
	IsDescending() bool
}

// Index describes an index on a table.
type Index interface {
	Stmt
	ColumnContainer

	HasType() bool
	HasName() bool
	HasSymbol() bool
	Name() string
	Reference() Reference
	SetReference(Reference) Index
	SetSymbol(string) Index
	SetType(IndexType) Index
	SetName(string) Index
	Symbol() string
	IsBtree() bool
	IsHash() bool
	IsPrimaryKey() bool
	IsNormal() bool
	IsUnique() bool
	IsFullText() bool
	IsSpatial() bool
	IsForeignKey() bool

	// Normalize returns normalized index. If a normalization was performed
	// and the index is modified, returns a new instance of the Table object
	// along with a true value as the second return value.
	// Otherwise, Normalize() returns the receiver unchanged, with a false
	// as the second return value.
	Normalize() (Index, bool)

	// Clone returns the clone index
	Clone() Index
}

// IndexKind describes the kind (purpose) of an index
type IndexKind int

// List of possible IndexKind.
const (
	IndexKindInvalid IndexKind = iota
	IndexKindPrimaryKey
	IndexKindNormal
	IndexKindUnique
	IndexKindFullText
	IndexKindSpatial
	IndexKindForeignKey
)

// IndexType describes the type (algorithm) used by the index.
type IndexType int

// List of possible index types
const (
	IndexTypeNone IndexType = iota
	IndexTypeBtree
	IndexTypeHash
)

// and index column specification may be
// name or name(length)
type indexColumn struct {
	name          string
	length        maybeString
	sortDirection IndexColumnSortDirection
}

type index struct {
	symbol  maybeString
	kind    IndexKind
	name    maybeString
	typ     IndexType
	table   string
	columns []IndexColumn
	// TODO Options.
	reference Reference
}

// Reference describes a possible reference from one table to another
type Reference interface {
	ColumnContainer

	ID() string
	String() string
	TableName() string
	OnDelete() ReferenceOption
	OnUpdate() ReferenceOption
	SetTableName(string) Reference
	SetMatch(ReferenceMatch) Reference
	SetOnDelete(ReferenceOption) Reference
	SetOnUpdate(ReferenceOption) Reference
	MatchFull() bool
	MatchPartial() bool
	MatchSimple() bool
}

type reference struct {
	tableName string
	columns   []IndexColumn
	match     ReferenceMatch
	onDelete  ReferenceOption
	onUpdate  ReferenceOption
}

// ReferenceMatch describes the mathing method of a reference
type ReferenceMatch int

// List of possible ReferenceMatch values
const (
	ReferenceMatchNone ReferenceMatch = iota
	ReferenceMatchFull
	ReferenceMatchPartial
	ReferenceMatchSimple
)

// ReferenceOption describes the actions that could be taken when
// a table/column referered by the reference has been deleted
type ReferenceOption int

// List of possible ReferenceOption values
const (
	ReferenceOptionNone ReferenceOption = iota
	ReferenceOptionRestrict
	ReferenceOptionCascade
	ReferenceOptionSetNull
	ReferenceOptionNoAction
)

// Table describes a table model
type Table interface {
	Stmt

	Name() string
	IsTemporary() bool
	SetTemporary(bool) Table
	IsIfNotExists() bool
	SetIfNotExists(bool) Table

	HasLikeTable() bool
	LikeTable() string
	SetLikeTable(string) Table

	AddColumn(TableColumn) Table
	Columns() chan TableColumn
	AddIndex(Index) Table
	Indexes() chan Index
	AddOption(TableOption) Table
	Options() chan TableOption

	LookupColumn(string) (TableColumn, bool)
	LookupColumnOrder(string) (int, bool)
	// LookupColumnBefore returns the table column before given column.
	// If the named column does not exist, or if the named column is
	// the first one, `(nil, false)` is returned
	LookupColumnBefore(string) (TableColumn, bool)

	LookupIndex(string) (Index, bool)

	// Normalize returns normalized table. If a normalization was performed
	// and the table is modified, returns a new instance of the Table object
	// along with a true value as the second return value.
	// Otherwise, Normalize() returns the receiver unchanged, with a false
	// as the second return value.
	Normalize() (Table, bool)
}

// TableOption describes a possible table option, such as `ENGINE=InnoDB`
type TableOption interface {
	Stmt
	Key() string
	Value() string
	NeedQuotes() bool
}

type table struct {
	mu                sync.RWMutex
	name              string
	temporary         bool
	ifnotexists       bool
	likeTable         maybeString
	columns           []TableColumn
	columnNameToIndex map[string]int
	indexes           []Index
	options           []TableOption
}

type tableopt struct {
	key        string
	value      string
	needQuotes bool
}

// NullState describes the possible NULL constraint of a column
type NullState int

// List of possible NullStates. NullStateNone specifies that there is
// no NULL constraint. NullStateNull explicitly specifies that the column
// may be NULL. NullStateNotNull specifies that the column may not be NULL
const (
	NullStateNone NullState = iota
	NullStateNull
	NullStateNotNull
)

// Length describes the possible length constraint of a column
type Length interface {
	HasDecimal() bool
	Decimal() string
	SetDecimal(string) Length
	Length() string
}

type length struct {
	decimals maybeString
	length   string
}

// TableColumn describes a model object that describes a column
// definition of a table
type TableColumn interface {
	Stmt

	TableID() string
	SetTableID(string) TableColumn

	Name() string
	Type() ColumnType
	SetType(ColumnType) TableColumn

	HasLength() bool
	Length() Length
	SetLength(Length) TableColumn
	HasCharacterSet() bool
	CharacterSet() string
	SetCharacterSet(string) TableColumn
	HasCollation() bool
	Collation() string
	SetCollation(string) TableColumn
	HasDefault() bool
	Default() string
	IsQuotedDefault() bool
	SetDefault(string, bool) TableColumn
	HasComment() bool
	Comment() string
	SetComment(string) TableColumn
	HasAutoUpdate() bool
	AutoUpdate() string
	SetAutoUpdate(string) TableColumn
	HasEnumValues() bool
	SetEnumValues([]string) TableColumn
	EnumValues() chan string
	HasSetValues() bool
	SetSetValues([]string) TableColumn
	SetValues() chan string

	NullState() NullState
	SetNullState(NullState) TableColumn

	IsAutoIncrement() bool
	SetAutoIncrement(bool) TableColumn
	IsBinary() bool
	SetBinary(bool) TableColumn
	IsKey() bool
	SetKey(bool) TableColumn
	IsPrimary() bool
	SetPrimary(bool) TableColumn
	IsUnique() bool
	SetUnique(bool) TableColumn
	IsUnsigned() bool
	SetUnsigned(bool) TableColumn
	IsZeroFill() bool
	SetZeroFill(bool) TableColumn

	// NativeLength returns the "native" size of a column type. This is the length used if you do not explicitly specify it.
	// Currently only supports numeric types, but may change later.
	NativeLength() Length

	// Normalize returns normalized column. If a normalization was performed
	// and the column is modified, returns a new instance of the Table object
	// along with a true value as the second return value.
	// Otherwise, Normalize() returns the receiver unchanged, with a false
	// as the second return value.
	//
	// Normalization is performed on numertic type display lengths, synonym
	// types, and NULL expressions
	Normalize() (TableColumn, bool)

	// Clone returns the cloned column
	Clone() TableColumn
}

type defaultValue struct {
	Valid  bool
	Value  string
	Quoted bool
}

type tablecol struct {
	tableID      string
	name         string
	typ          ColumnType
	length       Length
	nullstate    NullState
	charset      maybeString
	collation    maybeString
	defaultValue defaultValue
	comment      maybeString
	autoUpdate   maybeString
	enumValues   []string
	setValues    []string
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
	SetIfNotExists(bool) Database
}

type database struct {
	name        string
	ifnotexists bool
}
