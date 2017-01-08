package model

import (
	"bytes"

	"github.com/schemalex/schemalex/internal/errors"
	"github.com/schemalex/schemalex/internal/util"
)

func NewReference() Reference {
	return &reference{}
}

func (stmt *reference) AddColumns(l ...string) {
	stmt.columns = append(stmt.columns, l...)
}

func (stmt *reference) Columns() chan string {
	c := make(chan string, len(stmt.columns))
	for _, col := range stmt.columns {
		c <- col
	}
	close(c)
	return c
}

func (stmt *reference) TableName() string {
	return stmt.tableName
}

func (stmt *reference) MatchFull() bool {
	return stmt.match == ReferenceMatchFull
}

func (stmt *reference) MatchSimple() bool {
	return stmt.match == ReferenceMatchSimple
}

func (stmt *reference) MatchPartial() bool {
	return stmt.match == ReferenceMatchPartial
}

func (stmt *reference) OnDelete() ReferenceOption {
	return stmt.onDelete
}

func (stmt *reference) OnUpdate() ReferenceOption {
	return stmt.onUpdate
}

func (stmt *reference) SetMatch(v ReferenceMatch) {
	stmt.match = v
}

func (stmt *reference) SetOnDelete(v ReferenceOption) {
	stmt.onDelete = v
}

func (stmt *reference) SetOnUpdate(v ReferenceOption) {
	stmt.onUpdate = v
}

func (stmt *reference) SetTableName(v string) {
	stmt.tableName = v
}

func (r reference) String() string {
	var buf bytes.Buffer

	buf.WriteString("REFERENCES ")
	buf.WriteString(util.Backquote(r.TableName()))
	buf.WriteString(" (")

	ch := r.Columns()
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

	switch {
	case r.MatchFull():
		buf.WriteString(" MATCH FULL")
	case r.MatchPartial():
		buf.WriteString(" MATCH PARTIAL")
	case r.MatchSimple():
		buf.WriteString(" MATCH SIMPLE")
	}

	// we should really check for errors...
	writeReferenceOption(&buf, "ON DELETE", r.OnDelete())
	writeReferenceOption(&buf, "ON UPDATE", r.OnUpdate())

	return buf.String()
}


func writeReferenceOption(buf *bytes.Buffer, prefix string, opt ReferenceOption) error {
	if opt != ReferenceOptionNone {
		buf.WriteByte(' ')
		buf.WriteString(prefix)
		switch opt {
		case ReferenceOptionRestrict:
			buf.WriteString(" RESTRICT")
		case ReferenceOptionCascade:
			buf.WriteString(" CASCADE")
		case ReferenceOptionSetNull:
			buf.WriteString(" SET NULL")
		case ReferenceOptionNoAction:
			buf.WriteString(" NO ACTION")
		default:
			return errors.New("unknown reference option")
		}
	}
	return nil
}
