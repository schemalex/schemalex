package model

import (
	"bytes"
	"io"
	"strconv"

	"github.com/schemalex/schemalex/internal/util"
)

func NewLength(v string) Length {
	return &length{
		length: v,
	}
}

func (l *length) Decimal() string {
	return l.decimals.Value
}

func (l *length) HasDecimal() bool {
	return l.decimals.Valid
}

func (l *length) SetDecimal(v string) {
	l.decimals.Valid = true
	l.decimals.Value = v
}

func (l *length) Length() string {
	return l.length
}

func NewTableColumn(name string) TableColumn {
	return &tablecol{
		name: name,
	}
}

func (t *tablecol) ID() string {
	return "tablecol#" + t.name
}

func (t *tablecol) CharacterSet() string {
	return t.charset.Value
}

func (t *tablecol) Collation() string {
	return t.collation.Value
}

func (t *tablecol) Comment() string {
	return t.comment.Value
}

func (t *tablecol) Default() string {
	return t.defaultValue.Value
}

func (t *tablecol) HasCharacterSet() bool {
	return t.charset.Valid
}

func (t *tablecol) HasCollation() bool {
	return t.collation.Valid
}

func (t *tablecol) HasComment() bool {
	return t.comment.Valid
}

func (t *tablecol) HasDefault() bool {
	return t.defaultValue.Valid
}

func (t *tablecol) IsQuotedDefault() bool {
	return t.defaultValue.Quoted
}

func (t *tablecol) HasLength() bool {
	return t.length != nil
}

func (t *tablecol) IsAutoIncrement() bool {
	return t.autoincr
}

func (t *tablecol) IsBinary() bool {
	return t.binary
}

func (t *tablecol) IsKey() bool {
	return t.key
}

func (t *tablecol) IsPrimary() bool {
	return t.primary
}

func (t *tablecol) IsUnique() bool {
	return t.unique
}

func (t *tablecol) IsUnsigned() bool {
	return t.unsigned
}

func (t *tablecol) IsZeroFill() bool {
	return t.zerofill
}

func (t *tablecol) Length() Length {
	return t.length
}

func (t *tablecol) Name() string {
	return t.name
}

func (t *tablecol) NullState() NullState {
	return t.nullstate
}

func (t *tablecol) Type() ColumnType {
	return t.typ
}

func (t *tablecol) SetAutoIncrement(v bool) {
	t.autoincr = v
}

func (t *tablecol) SetBinary(v bool) {
	t.binary = v
}

func (t *tablecol) SetComment(v string) {
	t.comment.Valid = true
	t.comment.Value = v
}

func (t *tablecol) SetDefault(v string, quoted bool) {
	t.defaultValue.Valid = true
	t.defaultValue.Value = v
	t.defaultValue.Quoted = quoted
}

func (t *tablecol) SetKey(v bool) {
	t.key = v
}

func (t *tablecol) SetLength(v Length) {
	t.length = v
}

func (t *tablecol) SetNullState(v NullState) {
	t.nullstate = v
}

func (t *tablecol) SetPrimary(v bool) {
	t.primary = v
}

func (t *tablecol) SetType(v ColumnType) {
	t.typ = v
}

func (t *tablecol) SetUnique(v bool) {
	t.unique = v
}

func (t *tablecol) SetUnsigned(v bool) {
	t.unsigned = v
}

func (t *tablecol) SetZeroFill(v bool) {
	t.zerofill = v
}

func (t tablecol) WriteTo(dst io.Writer) (int64, error) {
	var buf bytes.Buffer

	buf.WriteString(util.Backquote(t.Name()))
	buf.WriteByte(' ')
	buf.WriteString(t.Type().String())

	if t.HasLength() {
		buf.WriteString(" (")
		l := t.Length()
		buf.WriteString(l.Length())
		if l.HasDecimal() {
			buf.WriteByte(',')
			buf.WriteString(l.Decimal())
		}
		buf.WriteByte(')')
	}

	if t.IsUnsigned() {
		buf.WriteString(" UNSIGNED")
	}

	if t.IsZeroFill() {
		buf.WriteString(" ZEROFILL")
	}

	if t.IsBinary() {
		buf.WriteString(" BINARY")
	}

	if t.HasCharacterSet() {
		buf.WriteString(" CHARACTER SET ")
		buf.WriteString(util.Backquote(t.CharacterSet()))
	}

	if t.HasCollation() {
		buf.WriteString(" COLLATE ")
		buf.WriteString(util.Backquote(t.Collation()))
	}

	if n := t.NullState(); n != NullStateNone {
		buf.WriteByte(' ')
		switch n {
		case NullStateNull:
			buf.WriteString("NULL")
		case NullStateNotNull:
			buf.WriteString("NOT NULL")
		}
	}

	if t.HasDefault() {
		buf.WriteString(" DEFAULT ")
		if t.IsQuotedDefault() {
			buf.WriteString(strconv.Quote(t.Default()))
		} else {
			buf.WriteString(t.Default())
		}
	}

	if t.IsAutoIncrement() {
		buf.WriteString(" AUTO_INCREMENT")
	}

	if t.IsUnique() {
		buf.WriteString(" UNIQUE KEY")
	}

	if t.IsPrimary() {
		buf.WriteString(" PRIMARY KEY")
	} else if t.IsKey() {
		buf.WriteString(" KEY")
	}

	if t.HasComment() {
		buf.WriteString(" '")
		buf.WriteString(t.Comment())
		buf.WriteByte('\'')
	}

	return buf.WriteTo(dst)
}
