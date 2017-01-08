package model

import "io"

type maybeString struct {
	Valid bool
	Value string
}

type WriteTo interface {
	WriteTo(io.Writer) (int64, error)
}

type ColumnContainer interface {
	AddColumns(...string)
	Columns() chan string
}

type Index interface {
	ColumnContainer
	WriteTo

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
