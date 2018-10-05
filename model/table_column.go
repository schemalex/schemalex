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

func (l *length) SetDecimal(v string) Length {
	l.decimals.Valid = true
	l.decimals.Value = v
	return l
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

func (t *tablecol) SetTableID(id string) TableColumn {
	t.tableID = id
	return t
}

func (t *tablecol) TableID() string {
	return t.tableID
}

func (t *tablecol) SetCharacterSet(s string) TableColumn {
	t.charset.Valid = true
	t.charset.Value = s
	return t
}

func (t *tablecol) SetCollation(s string) TableColumn {
	t.collation.Valid = true
	t.collation.Value = s
	return t
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

func (t *tablecol) SetAutoIncrement(v bool) TableColumn {
	t.autoincr = v
	return t
}

func (t *tablecol) SetBinary(v bool) TableColumn {
	t.binary = v
	return t
}

func (t *tablecol) SetComment(v string) TableColumn {
	t.comment.Valid = true
	t.comment.Value = v
	return t
}

func (t *tablecol) SetDefault(v string, quoted bool) TableColumn {
	t.defaultValue.Valid = true
	t.defaultValue.Value = v
	t.defaultValue.Quoted = quoted
	return t
}

func (t *tablecol) SetKey(v bool) TableColumn {
	t.key = v
	return t
}

func (t *tablecol) SetLength(v Length) TableColumn {
	t.length = v
	return t
}

func (t *tablecol) SetNullState(v NullState) TableColumn {
	t.nullstate = v
	return t
}

func (t *tablecol) SetPrimary(v bool) TableColumn {
	t.primary = v
	return t
}

func (t *tablecol) SetType(v ColumnType) TableColumn {
	t.typ = v
	return t
}

func (t *tablecol) SetUnique(v bool) TableColumn {
	t.unique = v
	return t
}

func (t *tablecol) SetUnsigned(v bool) TableColumn {
	t.unsigned = v
	return t
}

func (t *tablecol) SetZeroFill(v bool) TableColumn {
	t.zerofill = v
	return t
}

func (t *tablecol) HasAutoUpdate() bool {
	return t.autoUpdate.Valid
}

func (t *tablecol) SetAutoUpdate(s string) TableColumn {
	t.autoUpdate.Value = s
	t.autoUpdate.Valid = true
	return t
}

func (t *tablecol) AutoUpdate() string {
	return t.autoUpdate.Value
}

func (t *tablecol) HasEnumValues() bool {
	return len(t.enumValues) != 0
}

func (t *tablecol) SetEnumValues(enumValues []string) TableColumn {
	t.enumValues = enumValues
	return t
}

func (t *tablecol) EnumValues() chan string {
	ch := make(chan string, len(t.enumValues))
	for _, enumValue := range t.enumValues {
		ch <- enumValue
	}
	close(ch)
	return ch
}

func (t *tablecol) HasSetValues() bool {
	return len(t.setValues) != 0
}

func (t *tablecol) SetSetValues(setValues []string) TableColumn {
	t.setValues = setValues
	return t
}

func (t *tablecol) SetValues() chan string {
	ch := make(chan string, len(t.setValues))
	for _, setValue := range t.setValues {
		ch <- setValue
	}
	close(ch)
	return ch
}

func (t *tablecol) NativeLength() Length {
	// I referred to perl: SQL::Translator::Parser::MySQL#normalize_field https://metacpan.org/source/SQL::Translator::Parser::MySQL#L1072
	unsigned := 0
	if t.IsUnsigned() {
		unsigned++
	}
	var size int
	switch t.Type() {
	case ColumnTypeBool, ColumnTypeBoolean:
		// bool and boolean is tinyint(1)
		size = 1
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

func (t *tablecol) Normalize() (TableColumn, bool) {
	var clone bool
	var length Length
	var synonym ColumnType
	var removeQuotes bool
	var setDefaultNull bool

	if !t.HasLength() {
		if l := t.NativeLength(); l != nil {
			clone = true
			length = l
		}
	}

	if typ := t.Type(); typ.SynonymType() != typ {
		clone = true
		synonym = typ.SynonymType()
	}

	nullState := t.NullState()
	// remove null state if not `NOT NULL`
	// If none is specified, the column is treated as if NULL was specified.
	if nullState == NullStateNull {
		clone = true
		nullState = NullStateNone
	}

	if t.HasDefault() {
		switch t.Type() {
		case ColumnTypeTinyInt, ColumnTypeSmallInt,
			ColumnTypeMediumInt, ColumnTypeInt,
			ColumnTypeInteger, ColumnTypeBigInt,
			ColumnTypeFloat, ColumnTypeDouble,
			ColumnTypeDecimal, ColumnTypeNumeric, ColumnTypeReal:
			// If numeric type then trim quote
			if t.IsQuotedDefault() {
				clone = true
				removeQuotes = true
			}
		case ColumnTypeBool, ColumnTypeBoolean:
			switch t.Default() {
			case "TRUE":
				t.SetDefault("1", false)
			case "FALSE":
				t.SetDefault("0", false)
			}
		}
	} else {
		switch t.Type() {
		case ColumnTypeTinyText, ColumnTypeTinyBlob,
			ColumnTypeBlob, ColumnTypeText,
			ColumnTypeMediumBlob, ColumnTypeMediumText,
			ColumnTypeLongBlob, ColumnTypeLongText:
		default:
			// if nullable then set default null.
			if nullState != NullStateNotNull {
				clone = true
				setDefaultNull = true
			}
		}
	}

	// avoid cloning if we don't have to
	if !clone {
		return t, false
	}

	col := t.Clone()
	if length != nil {
		col.SetLength(length)
	}
	if synonym != ColumnTypeInvalid {
		col.SetType(synonym)
	}

	col.SetNullState(nullState)

	if removeQuotes {
		col.SetDefault(t.Default(), false)
	}

	if setDefaultNull {
		col.SetDefault("NULL", false)
	}
	return col, true
}

func (t *tablecol) Clone() TableColumn {
	col := &tablecol{}
	*col = *t
	return col
}
