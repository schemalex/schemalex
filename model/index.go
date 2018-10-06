package model

import (
	"crypto/sha256"
	"fmt"
)

// NewIndex creates a new index with the given index kind.
func NewIndex(kind IndexKind, table string) Index {
	return &index{
		kind:  kind,
		table: table,
	}
}

func (stmt *index) ID() string {
	// This is tricky. and index may or may not have a name. It would
	// have been so much easier if we did, but we don't, so we'll fake
	// something.
	//
	// In case we don't have a name, we need to know the table, the kind,
	// the type, // the column(s), and the reference(s).
	name := "index"
	if stmt.HasName() {
		name = name + "#" + stmt.Name()
	}
	h := sha256.New()

	sym := "none"
	if stmt.HasSymbol() {
		sym = stmt.Symbol()
	}

	fmt.Fprintf(h,
		"%s.%s.%s.%s",
		stmt.table,
		sym,
		stmt.kind,
		stmt.typ,
	)
	for col := range stmt.Columns() {
		fmt.Fprintf(h, ".")
		fmt.Fprintf(h, "%s", col.ID())
	}
	if stmt.reference != nil {
		fmt.Fprintf(h, ".")
		fmt.Fprintf(h, stmt.reference.ID())
	}
	return fmt.Sprintf("%s#%x", name, h.Sum(nil))
}

func (stmt *index) AddColumns(l ...IndexColumn) {
	stmt.columns = append(stmt.columns, l...)
}

func (stmt *index) Columns() chan IndexColumn {
	c := make(chan IndexColumn, len(stmt.columns))
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

func (stmt *index) SetReference(r Reference) Index {
	stmt.reference = r
	return stmt
}

func (stmt *index) SetName(s string) Index {
	stmt.name.Valid = true
	stmt.name.Value = s
	return stmt
}

func (stmt *index) SetSymbol(s string) Index {
	stmt.symbol.Valid = true
	stmt.symbol.Value = s
	return stmt
}

func (stmt *index) HasType() bool {
	return stmt.typ != IndexTypeNone
}

func (stmt *index) SetType(typ IndexType) Index {
	stmt.typ = typ
	return stmt
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

func (stmt *index) IsForeignKey() bool {
	return stmt.kind == IndexKindForeignKey
}

func (stmt *index) Normalize() (Index, bool) {
	return stmt, false
}

func (stmt *index) Clone() Index {
	newindex := &index{}
	*newindex = *stmt
	return newindex
}

func NewIndexColumn(name string) IndexColumn {
	return &indexColumn{
		name: name,
	}
}

func (col *indexColumn) ID() string {
	if col.HasLength() {
		return "index_column#" + col.Name() + "-" + col.Length()
	}
	return "index_column#" + col.Name()
}

func (col *indexColumn) Name() string {
	return col.name
}

func (col *indexColumn) HasLength() bool {
	return col.length.Valid
}

func (col *indexColumn) Length() string {
	return col.length.Value
}

func (col *indexColumn) SetLength(s string) IndexColumn {
	col.length.Valid = true
	col.length.Value = s
	return col
}

func (col *indexColumn) SetSortDirection(v IndexColumnSortDirection) {
	col.sortDirection = v
}

func (col *indexColumn) HasSortDirection() bool {
	return col.sortDirection != SortDirectionNone
}

func (col *indexColumn) IsAscending() bool {
	return col.sortDirection == SortDirectionAscending
}

func (col *indexColumn) IsDescending() bool {
	return col.sortDirection == SortDirectionDescending
}

