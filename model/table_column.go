package model

import (
	"strconv"
)

// NewLength creates a new Length which describes the
// length of a column
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

// NewTableColumn creates a new TableColumn with the given name
func NewTableColumn(name string) TableColumn {
	return &tablecol{
		name: name,
	}
}

func (t *tablecol) ID() string {
	return "tablecol#" + t.name
}

func (t *tablecol) SetCharacterSet(s string) {
	t.charset.Valid = true
	t.charset.Value = s
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

func (t *tablecol) HasAutoUpdate() bool {
	return t.autoUpdate.Valid
}

func (t *tablecol) SetAutoUpdate(s string) {
	t.autoUpdate.Value = s
	t.autoUpdate.Valid = true
}

func (t *tablecol) AutoUpdate() string {
	return t.autoUpdate.Value
}

func (t *tablecol) NativeLength() Length {
	// I referred to perl: SQL::Translator::Parser::MySQL#normalize_field https://metacpan.org/source/SQL::Translator::Parser::MySQL#L1072
	unsigned := 0
	if t.IsUnsigned() {
		unsigned++
	}
	var size int
	switch t.Type() {
	case ColumnTypeTinyInt:
		size = 4 - unsigned
	case ColumnTypeSmallInt:
		size = 6 - unsigned
	case ColumnTypeMediumInt:
		size = 9 - unsigned
	case ColumnTypeInt, ColumnTypeInteger:
		size = 11 - unsigned
	case ColumnTypeBigInt:
		size = 20
	case ColumnTypeDecimal, ColumnTypeNumeric:
		// DECIMAL(M) means DECIMAL(M,0)
		// The default value of M is 10.
		// https://dev.mysql.com/doc/refman/5.6/en/fixed-point-types.html
		l := NewLength("10")
		l.SetDecimal("0")
		return l
	default:
		return nil
	}

	return NewLength(strconv.Itoa(size))
}

func (t *tablecol) Normalize() TableColumn {
	col := t.clone()
	if !t.HasLength() {
		if length := t.NativeLength(); length != nil {
			col.SetLength(length)
		}
	}

	if synonym := t.Type().SynonymType(); synonym != t.Type() {
		col.SetType(synonym)
	}

	col.normalizeNullExpression()
	return col
}

func (t *tablecol) normalizeNullExpression() {
	// remove null state if not `NOT NULL`
	// If none is specified, the column is treated as if NULL was specified.
	if t.NullState() == NullStateNull {
		t.SetNullState(NullStateNone)
	}
	if t.HasDefault() {
		switch t.Type() {
		case ColumnTypeTinyInt, ColumnTypeSmallInt,
			ColumnTypeMediumInt, ColumnTypeInt,
			ColumnTypeInteger, ColumnTypeBigInt,
			ColumnTypeFloat, ColumnTypeDouble,
			ColumnTypeDecimal, ColumnTypeNumeric, ColumnTypeReal:
			// If numeric type then trim quote
			t.SetDefault(t.Default(), false)
		}
	} else {
		switch t.Type() {
		case ColumnTypeTinyText, ColumnTypeTinyBlob,
			ColumnTypeBlob, ColumnTypeText,
			ColumnTypeMediumBlob, ColumnTypeMediumText,
			ColumnTypeLongBlob, ColumnTypeLongText:
		default:
			// if nullable then set default null.
			if t.NullState() != NullStateNotNull {
				t.SetDefault("NULL", false)
			}
		}
	}
}

func (t *tablecol) clone() *tablecol {
	col := &tablecol{}
	*col = *t
	return col
}
