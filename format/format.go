package format

import (
	"bytes"
	"io"

	"github.com/schemalex/schemalex/internal/errors"
	"github.com/schemalex/schemalex/internal/util"
	"github.com/schemalex/schemalex/model"
)

// SQL takes an arbitrary `model.*` object and formats it as SQL,
// writing its result to `dst`
func SQL(dst io.Writer, v interface{}) error {
	switch v.(type) {
	case model.ColumnType:
		return formatColumnType(dst, v.(model.ColumnType))
	case model.Database:
		return formatDatabase(dst, v.(model.Database))
	case model.Stmts:
		for _, s := range v.(model.Stmts) {
			if err := SQL(dst, s); err != nil {
				return err
			}
		}
		return nil
	case model.Table:
		return formatTable(dst, v.(model.Table))
	case model.TableColumn:
		return formatTableColumn(dst, v.(model.TableColumn))
	case model.TableOption:
		return formatTableOption(dst, v.(model.TableOption))
	case model.Index:
		return formatIndex(dst, v.(model.Index))
	case model.Reference:
		return formatReference(dst, v.(model.Reference))
	default:
		return errors.New("unsupported model type")
	}
}

func formatDatabase(dst io.Writer, d model.Database) error {
	var buf bytes.Buffer
	buf.WriteString("CREATE DATABASE")
	if d.IsIfNotExists() {
		buf.WriteString(" IF NOT EXISTS")
	}
	buf.WriteByte(' ')
	buf.WriteString(util.Backquote(d.Name()))
	buf.WriteByte(';')

	if _, err := buf.WriteTo(dst); err != nil {
		return err
	}
	return nil
}

func formatTableOption(dst io.Writer, option model.TableOption) error {
	var buf bytes.Buffer
	buf.WriteString(option.Key())
	buf.WriteString(" = ")
	if option.NeedQuotes() {
		buf.WriteByte('\'')
		buf.WriteString(option.Value())
		buf.WriteByte('\'')
	} else {
		buf.WriteString(option.Value())
	}

	if _, err := buf.WriteTo(dst); err != nil {
		return err
	}
	return nil
}

func formatTable(dst io.Writer, table model.Table) error {
	var buf bytes.Buffer

	buf.WriteString("CREATE")
	if table.IsTemporary() {
		buf.WriteString(" TEMPORARY")
	}

	buf.WriteString(" TABLE")
	if table.IsIfNotExists() {
		buf.WriteString(" IF NOT EXISTS")
	}

	buf.WriteByte(' ')
	buf.WriteString(util.Backquote(table.Name()))

	if table.HasLikeTable() {
		buf.WriteString(" LIKE ")
		buf.WriteString(util.Backquote(table.LikeTable()))
	} else {
		buf.WriteString(" (")

		colch := table.Columns()
		idxch := table.Indexes()
		colchmax := len(colch)
		idxchmax := len(idxch)

		var i int
		for col := range colch {
			buf.WriteByte('\n')
			if err := formatTableColumn(&buf, col); err != nil {
				return err
			}
			if i < colchmax-1 || idxchmax > 0 {
				buf.WriteByte(',')
			}
			i++
		}

		i = 0
		for idx := range idxch {
			buf.WriteByte('\n')
			if err := formatIndex(&buf, idx); err != nil {
				return err
			}
			if i < idxchmax-1 {
				buf.WriteByte(',')
			}
			i++
		}

		buf.WriteString("\n)")

		optch := table.Options()
		if l := len(optch); l > 0 {
			buf.WriteByte(' ')
			var i int
			for option := range optch {
				if err := formatTableOption(&buf, option); err != nil {
					return err
				}

				if i < l-1 {
					buf.WriteString(", ")
				}
				i++
			}
		}
	}

	if _, err := buf.WriteTo(dst); err != nil {
		return err
	}
	return nil
}

func formatColumnType(dst io.Writer, col model.ColumnType) error {
	if col <= model.ColumnTypeInvalid || col >= model.ColumnTypeMax {
		return errors.New(`invalid column type`)
	}

	if _, err := io.WriteString(dst, col.String()); err != nil {
		return err
	}
	return nil
}

func formatTableColumn(dst io.Writer, col model.TableColumn) error {
	var buf bytes.Buffer

	buf.WriteString(util.Backquote(col.Name()))
	buf.WriteByte(' ')

	if err := formatColumnType(&buf, col.Type()); err != nil {
		return err
	}

	if col.HasEnumValues() {
		buf.WriteString(" (")
		isFirst := true
		for enumValue := range col.EnumValues() {
			if !isFirst {
				buf.WriteString(", ")
			}
			buf.WriteByte('\'')
			buf.WriteString(enumValue)
			buf.WriteByte('\'')
			isFirst = false
		}
		buf.WriteByte(')')
	}

	if col.HasSetValues() {
		buf.WriteString(" (")
		isFirst := true
		for setValue := range col.SetValues() {
			if !isFirst {
				buf.WriteString(", ")
			}
			buf.WriteByte('\'')
			buf.WriteString(setValue)
			buf.WriteByte('\'')
			isFirst = false
		}
		buf.WriteByte(')')
	}

	if col.HasLength() {
		buf.WriteString(" (")
		l := col.Length()
		buf.WriteString(l.Length())
		if l.HasDecimal() {
			buf.WriteByte(',')
			buf.WriteString(l.Decimal())
		}
		buf.WriteByte(')')
	}

	if col.IsUnsigned() {
		buf.WriteString(" UNSIGNED")
	}

	if col.IsZeroFill() {
		buf.WriteString(" ZEROFILL")
	}

	if col.IsBinary() {
		buf.WriteString(" BINARY")
	}

	if col.HasCharacterSet() {
		buf.WriteString(" CHARACTER SET ")
		buf.WriteString(util.Backquote(col.CharacterSet()))
	}

	if col.HasCollation() {
		buf.WriteString(" COLLATE ")
		buf.WriteString(util.Backquote(col.Collation()))
	}

	if col.HasAutoUpdate() {
		buf.WriteString(" ON UPDATE ")
		buf.WriteString(col.AutoUpdate())
	}

	if n := col.NullState(); n != model.NullStateNone {
		buf.WriteByte(' ')
		switch n {
		case model.NullStateNull:
			buf.WriteString("NULL")
		case model.NullStateNotNull:
			buf.WriteString("NOT NULL")
		}
	}

	if col.HasDefault() {
		buf.WriteString(" DEFAULT ")
		if col.IsQuotedDefault() {
			buf.WriteByte('\'')
			buf.WriteString(col.Default())
			buf.WriteByte('\'')
		} else {
			buf.WriteString(col.Default())
		}
	}

	if col.IsAutoIncrement() {
		buf.WriteString(" AUTO_INCREMENT")
	}

	if col.IsUnique() {
		buf.WriteString(" UNIQUE KEY")
	}

	if col.IsPrimary() {
		buf.WriteString(" PRIMARY KEY")
	} else if col.IsKey() {
		buf.WriteString(" KEY")
	}

	if col.HasComment() {
		buf.WriteString(" COMMENT '")
		buf.WriteString(col.Comment())
		buf.WriteByte('\'')
	}

	if _, err := buf.WriteTo(dst); err != nil {
		return err
	}
	return nil
}

func formatIndex(dst io.Writer, index model.Index) error {
	var buf bytes.Buffer

	if index.HasSymbol() {
		buf.WriteString("CONSTRAINT ")
		buf.WriteString(util.Backquote(index.Symbol()))
		buf.WriteByte(' ')
	}

	switch {
	case index.IsPrimaryKey():
		buf.WriteString("PRIMARY KEY")
	case index.IsNormal():
		buf.WriteString("INDEX")
	case index.IsUnique():
		buf.WriteString("UNIQUE INDEX")
	case index.IsFullText():
		buf.WriteString("FULLTEXT INDEX")
	case index.IsSpatial():
		buf.WriteString("SPATIAL INDEX")
	case index.IsForeginKey():
		buf.WriteString("FOREIGN KEY")
	}

	if index.HasName() {
		buf.WriteByte(' ')
		buf.WriteString(util.Backquote(index.Name()))
	}

	switch {
	case index.IsBtree():
		buf.WriteString(" USING BTREE")
	case index.IsHash():
		buf.WriteString(" USING HASH")
	}

	buf.WriteString(" (")
	ch := index.Columns()
	lch := len(ch)
	if lch == 0 {
		return errors.New(`no columns in index`)
	}

	var i int
	for col := range ch {
		buf.WriteString(util.Backquote(col.Name()))
		if col.HasLength() {
			buf.WriteByte('(')
			buf.WriteString(col.Length())
			buf.WriteByte(')')
		}
		if i < lch-1 {
			buf.WriteString(", ")
		}
		i++
	}
	buf.WriteByte(')')

	if ref := index.Reference(); ref != nil {
		buf.WriteByte(' ')
		if err := formatReference(&buf, ref); err != nil {
			return err
		}
	}

	if _, err := buf.WriteTo(dst); err != nil {
		return err
	}
	return nil
}

func formatReference(dst io.Writer, r model.Reference) error {
	var buf bytes.Buffer

	buf.WriteString("REFERENCES ")
	buf.WriteString(util.Backquote(r.TableName()))
	buf.WriteString(" (")

	ch := r.Columns()
	lch := len(ch)
	var i int
	for col := range ch {
		buf.WriteString(util.Backquote(col.Name()))
		if col.HasLength() {
			buf.WriteByte('(')
			buf.WriteString(col.Length())
			buf.WriteByte(')')
		}
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

	if _, err := buf.WriteTo(dst); err != nil {
		return err
	}
	return nil
}

func writeReferenceOption(buf *bytes.Buffer, prefix string, opt model.ReferenceOption) error {
	if opt != model.ReferenceOptionNone {
		buf.WriteByte(' ')
		buf.WriteString(prefix)
		switch opt {
		case model.ReferenceOptionRestrict:
			buf.WriteString(" RESTRICT")
		case model.ReferenceOptionCascade:
			buf.WriteString(" CASCADE")
		case model.ReferenceOptionSetNull:
			buf.WriteString(" SET NULL")
		case model.ReferenceOptionNoAction:
			buf.WriteString(" NO ACTION")
		default:
			return errors.New("unknown reference option")
		}
	}
	return nil
}
