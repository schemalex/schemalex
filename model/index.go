package model

import (
	"bytes"
	"io"

	"github.com/schemalex/schemalex/internal/util"
)

func NewIndex(kind IndexKind) Index {
	return &index{
		kind: kind,
	}
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
	case IndexKindSpatial:
		return "SPARTIAL INDEX"
	case IndexKindForeignKey:
		return "FOREIGN KEY"
	default:
		return "(invalid)"
	}
}

func (stmt *index) String() string {
	var buf bytes.Buffer
	stmt.WriteTo(&buf)
	return buf.String()
}

func (stmt *index) WriteTo(dst io.Writer) (int64, error) {
	var buf bytes.Buffer

	if stmt.HasSymbol() {
		buf.WriteString("CONSTRAINT ")
		buf.WriteString(util.Backquote(stmt.Symbol()))
		buf.WriteByte(' ')
	}

	buf.WriteString(stmt.kind.String())

	if stmt.HasName() {
		buf.WriteByte(' ')
		buf.WriteString(util.Backquote(stmt.Name()))
	}

	if str := stmt.typ.String(); str != "" {
		buf.WriteByte(' ')
		buf.WriteString(str)
	}

	buf.WriteString(" (")
	ch := stmt.Columns()
	lch := len(ch)
	var i int
	for col := range ch {
		buf.WriteString(util.Backquote(col))
		if i < lch-1 {
			buf.WriteString(", ")
		}
		i++
	}
	buf.WriteByte(')')

	if ref := stmt.Reference(); ref != nil {
		buf.WriteByte(' ')
		buf.WriteString(ref.String())
	}

	return buf.WriteTo(dst)
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
		return "(invalid)"
	}
}
