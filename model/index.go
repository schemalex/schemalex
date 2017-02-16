package model

import (
	"crypto/sha256"
	"fmt"
	"io"
)

// NewIndex creates a new index with the given index kind.
func NewIndex(kind IndexKind) Index {
	return &index{
		kind: kind,
	}
}

func (stmt *index) ID() string {
	// This is tricky. and index may or may not have a name. It would
	// have been so much easier if we did, but we don't, so we'll fake
	// something
	name := "index"
	if stmt.HasName() {
		name = name + "#" + stmt.Name()
	}
	h := sha256.New()
	io.WriteString(h, fmt.Sprintf("%v, %v, %v, %v, %v, %v", stmt.symbol, stmt.kind, stmt.name, stmt.typ, stmt.columns, stmt.reference))
	return fmt.Sprintf("%s#%x", name, h.Sum(nil))
}

func (stmt *index) AddColumns(l ...string) {
	stmt.columns = append(stmt.columns, l...)
}

func (stmt *index) Columns() chan string {
	c := make(chan string, len(stmt.columns))
	for _, col := range stmt.columns {
		c <- col
	}
	close(c)
	return c
}

func (stmt *index) Reference() Reference {
	return stmt.reference
}

func (stmt *index) Name() string {
	return stmt.name.Value
}

func (stmt *index) Symbol() string {
	return stmt.symbol.Value
}

func (stmt *index) HasName() bool {
	return stmt.name.Valid
}

func (stmt *index) HasSymbol() bool {
	return stmt.symbol.Valid
}

func (stmt *index) SetReference(r Reference) {
	stmt.reference = r
}

func (stmt *index) SetName(s string) {
	stmt.name.Valid = true
	stmt.name.Value = s
}

func (stmt *index) SetSymbol(s string) {
	stmt.symbol.Valid = true
	stmt.symbol.Value = s
}

func (stmt *index) SetType(typ IndexType) {
	stmt.typ = typ
}

func (stmt *index) IsBtree() bool {
	return stmt.typ == IndexTypeBtree
}

func (stmt *index) IsHash() bool {
	return stmt.typ == IndexTypeHash
}

func (stmt *index) IsPrimaryKey() bool {
	return stmt.kind == IndexKindPrimaryKey
}

func (stmt *index) IsNormal() bool {
	return stmt.kind == IndexKindNormal
}

func (stmt *index) IsUnique() bool {
	return stmt.kind == IndexKindUnique
}

func (stmt *index) IsFullText() bool {
	return stmt.kind == IndexKindFullText
}

func (stmt *index) IsSpatial() bool {
	return stmt.kind == IndexKindSpatial
}

func (stmt *index) IsForeginKey() bool {
	return stmt.kind == IndexKindForeignKey
}
