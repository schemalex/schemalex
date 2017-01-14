package model

import (
	"bytes"

	"github.com/schemalex/schemalex/internal/errors"
	"github.com/schemalex/schemalex/internal/util"
)

// NewReference creates a reference constraint
func NewReference() Reference {
	return &reference{}
}

func (r *reference) AddColumns(l ...string) {
	r.columns = append(r.columns, l...)
}

func (r *reference) Columns() chan string {
	c := make(chan string, len(r.columns))
	for _, col := range r.columns {
		c <- col
	}
	close(c)
	return c
}

func (r *reference) TableName() string {
	return r.tableName
}

func (r *reference) MatchFull() bool {
	return r.match == ReferenceMatchFull
}

func (r *reference) MatchSimple() bool {
	return r.match == ReferenceMatchSimple
}

func (r *reference) MatchPartial() bool {
	return r.match == ReferenceMatchPartial
}

func (r *reference) OnDelete() ReferenceOption {
	return r.onDelete
}

func (r *reference) OnUpdate() ReferenceOption {
	return r.onUpdate
}

func (r *reference) SetMatch(v ReferenceMatch) {
	r.match = v
}

func (r *reference) SetOnDelete(v ReferenceOption) {
	r.onDelete = v
}

func (r *reference) SetOnUpdate(v ReferenceOption) {
	r.onUpdate = v
}

func (r *reference) SetTableName(v string) {
	r.tableName = v
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
