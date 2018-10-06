package format

import (
	"bytes"
	"io"

	"github.com/schemalex/schemalex/internal/errors"
	"github.com/schemalex/schemalex/internal/util"
	"github.com/schemalex/schemalex/model"
)

type fmtCtx struct {
	curIndent string
	dst       io.Writer
	indent    string
}

func newFmtCtx(dst io.Writer) *fmtCtx {
	return &fmtCtx{
		dst: dst,
	}
}

func (ctx *fmtCtx) clone() *fmtCtx {
	return &fmtCtx{
		curIndent: ctx.curIndent,
		dst:       ctx.dst,
		indent:    ctx.indent,
	}
}

// SQL takes an arbitrary `model.*` object and formats it as SQL,
// writing its result to `dst`
func SQL(dst io.Writer, v interface{}, options ...Option) error {
	ctx := newFmtCtx(dst)
	for _, o := range options {
		switch o.Name() {
		case optkeyIndent:
			ctx.indent = o.Value().(string)
		}
	}

	return format(ctx, v)
}

func format(ctx *fmtCtx, v interface{}) error {
	switch v.(type) {
	case model.ColumnType:
		return formatColumnType(ctx, v.(model.ColumnType))
	case model.Database:
		return formatDatabase(ctx, v.(model.Database))
	case model.Stmts:
		for _, s := range v.(model.Stmts) {
			if err := format(ctx, s); err != nil {
				return err
			}
		}
		return nil
	case model.Table:
		return formatTable(ctx, v.(model.Table))
	case model.TableColumn:
		return formatTableColumn(ctx, v.(model.TableColumn))
	case model.TableOption:
		return formatTableOption(ctx, v.(model.TableOption))
	case model.Index:
		return formatIndex(ctx, v.(model.Index))
	case model.Reference:
		return formatReference(ctx, v.(model.Reference))
	default:
		return errors.New("unsupported model type")
	}
}

func formatDatabase(ctx *fmtCtx, d model.Database) error {
	var buf bytes.Buffer
	buf.WriteString("CREATE DATABASE")
	if d.IsIfNotExists() {
		buf.WriteString(" IF NOT EXISTS")
	}
	buf.WriteByte(' ')
	buf.WriteString(util.Backquote(d.Name()))
	buf.WriteByte(';')

	if _, err := buf.WriteTo(ctx.dst); err != nil {
		return err
	}
	return nil
}

func formatTableOption(ctx *fmtCtx, option model.TableOption) error {
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

	if _, err := buf.WriteTo(ctx.dst); err != nil {
		return err
	}
	return nil
}

func formatTable(ctx *fmtCtx, table model.Table) error {
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

		newctx := ctx.clone()
		newctx.curIndent = newctx.indent + newctx.curIndent
		newctx.dst = &buf

		buf.WriteString(" (")

		colch := table.Columns()
		idxch := table.Indexes()
		colchmax := len(colch)
		idxchmax := len(idxch)

		var i int
		for col := range colch {
			buf.WriteByte('\n')
			if err := formatTableColumn(newctx, col); err != nil {
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
			if err := formatIndex(newctx, idx); err != nil {
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
				if err := formatTableOption(newctx, option); err != nil {
					return err
				}

				if i < l-1 {
					buf.WriteString(", ")
				}
				i++
			}
		}
	}

	if _, err := buf.WriteTo(ctx.dst); err != nil {
		return err
	}
	return nil
}

func formatColumnType(ctx *fmtCtx, col model.ColumnType) error {
	if col <= model.ColumnTypeInvalid || col >= model.ColumnTypeMax {
		return errors.New(`invalid column type`)
	}

	if _, err := io.WriteString(ctx.dst, col.String()); err != nil {
		return err
	}

	return nil
}

func formatTableColumn(ctx *fmtCtx, col model.TableColumn) error {
	var buf bytes.Buffer

	buf.WriteString(ctx.curIndent)
	buf.WriteString(util.Backquote(col.Name()))
	buf.WriteByte(' ')

	newctx := ctx.clone()
	newctx.curIndent = ""
	newctx.dst = &buf
	if err := formatColumnType(newctx, col.Type()); err != nil {
		return err
	}

	switch col.Type() {
	case model.ColumnTypeEnum:
		buf.WriteString(" (")
		for enumValue := range col.EnumValues() {
			buf.WriteByte('\'')
			buf.WriteString(enumValue)
			buf.WriteByte('\'')
			buf.WriteByte(',')
		}
		buf.Truncate(buf.Len() - 1)
		buf.WriteByte(')')
	case model.ColumnTypeSet:
		buf.WriteString(" (")
		for setValue := range col.SetValues() {
			buf.WriteByte('\'')
			buf.WriteString(setValue)
			buf.WriteByte('\'')
			buf.WriteByte(',')
		}
		buf.Truncate(buf.Len() - 1)
		buf.WriteByte(')')
	default:
		if col.HasLength() {
			l := col.Length()
			buf.WriteString(" (")
			buf.WriteString(l.Length())
			if l.HasDecimal() {
				buf.WriteByte(',')
				buf.WriteString(l.Decimal())
			}
			buf.WriteByte(')')
		}
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

	if _, err := buf.WriteTo(ctx.dst); err != nil {
		return err
	}
	return nil
}

func formatIndex(ctx *fmtCtx, index model.Index) error {
	var buf bytes.Buffer

	buf.WriteString(ctx.curIndent)
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
	case index.IsForeignKey():
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
		if col.HasSortDirection() {
			if col.IsAscending() {
				buf.WriteString(" ASC")
			} else {
				buf.WriteString(" DESC")
			}
		}

		if i < lch-1 {
			buf.WriteString(", ")
		}
		i++
	}
	buf.WriteByte(')')

	if ref := index.Reference(); ref != nil {
		newctx := ctx.clone()
		newctx.dst = &buf

		buf.WriteByte(' ')
		if err := formatReference(newctx, ref); err != nil {
			return err
		}
	}

	if _, err := buf.WriteTo(ctx.dst); err != nil {
		return err
	}
	return nil
}

func formatReference(ctx *fmtCtx, r model.Reference) error {
	var buf bytes.Buffer

	buf.WriteString(ctx.curIndent)
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

	if _, err := buf.WriteTo(ctx.dst); err != nil {
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
