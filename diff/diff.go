package diff

import (
	"bytes"
	"io"
	"reflect"

	"github.com/deckarep/golang-set"
	"github.com/schemalex/schemalex"
	"github.com/schemalex/schemalex/format"
	"github.com/schemalex/schemalex/internal/errors"
	"github.com/schemalex/schemalex/model"
)

type Option interface {
	Name() string
	Value() interface{}
}

type option struct {
	name  string
	value interface{}
}

func (o option) Name() string       { return o.name }
func (o option) Value() interface{} { return o.value }

func WithParser(p *schemalex.Parser) Option {
	return &option{
		name:  "parser",
		value: p,
	}
}

func WithTransaction(b bool) Option {
	return &option{
		name:  "transaction",
		value: b,
	}
}

type diffCtx struct {
	fromSet mapset.Set
	toSet   mapset.Set
	from    model.Stmts
	to      model.Stmts
}

func newDiffCtx(from, to model.Stmts) *diffCtx {
	fromSet := mapset.NewSet()
	for _, stmt := range from {
		if cs, ok := stmt.(model.Table); ok {
			fromSet.Add(cs.ID())
		}
	}
	toSet := mapset.NewSet()
	for _, stmt := range to {
		if cs, ok := stmt.(model.Table); ok {
			toSet.Add(cs.ID())
		}
	}

	return &diffCtx{
		fromSet: fromSet,
		toSet:   toSet,
		from:    from,
		to:      to,
	}
}

func Statements(dst io.Writer, from, to model.Stmts, options ...Option) error {
	var txn bool
	for _, o := range options {
		switch o.Name() {
		case "transaction":
			txn = o.Value().(bool)
		}
	}

	ctx := newDiffCtx(from, to)

	var procs = []func(*diffCtx, io.Writer) (int64, error){
		dropTables,
		createTables,
		alterTables,
	}

	var buf bytes.Buffer
	if txn {
		buf.WriteString("\nBEGIN;\n\nSET FOREIGN_KEY_CHECKS = 0;")
	}

	for _, p := range procs {
		var pbuf bytes.Buffer
		n, err := p(ctx, &pbuf)
		if err != nil {
			return errors.Wrap(err, `failed to produce diff`)
		}
		if txn && n > 0 || !txn && buf.Len() > 0 && n > 0 {
			buf.WriteString("\n\n")
		}
		pbuf.WriteTo(&buf)
	}
	if txn {
		buf.WriteString("\n\nSET FOREIGN_KEY_CHECKS = 1;\n\nCOMMIT;")
	}

	if _, err := buf.WriteTo(dst); err != nil {
		return errors.Wrap(err, `failed to write diff`)
	}
	return nil
}

func Strings(dst io.Writer, from, to string, options ...Option) error {
	var p *schemalex.Parser
	for _, o := range options {
		switch o.Name() {
		case "parser":
			p = o.Value().(*schemalex.Parser)
		}
	}
	if p == nil {
		p = schemalex.New()
	}

	stmts1, err := p.ParseString(from)
	if err != nil {
		return errors.Wrapf(err, `failed to parse "from" %s`, from)
	}

	stmts2, err := p.ParseString(to)
	if err != nil {
		return errors.Wrapf(err, `failed to parse "to" %s`, to)
	}

	return Statements(dst, stmts1, stmts2, options...)
}

func Files(dst io.Writer, from, to string, options ...Option) error {
	var p *schemalex.Parser
	for _, o := range options {
		switch o.Name() {
		case "parser":
			p = o.Value().(*schemalex.Parser)
		}
	}
	if p == nil {
		p = schemalex.New()
	}
	stmts1, err := p.ParseFile(from)
	if err != nil {
		return errors.Wrapf(err, `failed to open "from" file %s`, from)
	}

	stmts2, err := p.ParseFile(to)
	if err != nil {
		return errors.Wrapf(err, `failed to open "to" file %s`, to)
	}

	return Statements(dst, stmts1, stmts2, options...)
}

func dropTables(ctx *diffCtx, dst io.Writer) (int64, error) {
	var buf bytes.Buffer
	ids := ctx.fromSet.Difference(ctx.toSet)
	for i, id := range ids.ToSlice() {
		if i > 0 {
			buf.WriteByte('\n')
		}

		stmt, ok := ctx.from.Lookup(id.(string))
		if !ok {
			return 0, errors.Errorf(`failed to lookup table %s`, id)
		}

		table, ok := stmt.(model.Table)
		if !ok {
			return 0, errors.Errorf(`lookup failed: %s is not a model.Table`, id)
		}
		buf.WriteString("DROP TABLE `")
		buf.WriteString(table.Name())
		buf.WriteString("`;")
	}

	return buf.WriteTo(dst)
}

func createTables(ctx *diffCtx, dst io.Writer) (int64, error) {
	var buf bytes.Buffer

	ids := ctx.toSet.Difference(ctx.fromSet)
	for _, id := range ids.ToSlice() {
		// Lookup the corresponding statement, and add its SQL
		stmt, ok := ctx.to.Lookup(id.(string))
		if !ok {
			return 0, errors.Errorf(`failed to lookup table %s`, id)
		}

		if buf.Len() > 0 {
			buf.WriteByte('\n')
		}

		if err := format.SQL(&buf, stmt); err != nil {
			return 0, err
		}
		buf.WriteByte(';')
	}
	return buf.WriteTo(dst)
}

type alterCtx struct {
	fromColumns mapset.Set
	toColumns   mapset.Set
	fromIndexes mapset.Set
	toIndexes   mapset.Set
	from        model.Table
	to          model.Table
}

func newAlterCtx(from, to model.Table) *alterCtx {
	fromColumns := mapset.NewSet()
	for col := range from.Columns() {
		fromColumns.Add(col.ID())
	}

	toColumns := mapset.NewSet()
	for col := range to.Columns() {
		toColumns.Add(col.ID())
	}

	fromIndexes := mapset.NewSet()
	for idx := range from.Indexes() {
		fromIndexes.Add(idx.ID())
	}

	toIndexes := mapset.NewSet()
	for idx := range to.Indexes() {
		toIndexes.Add(idx.ID())
	}

	return &alterCtx{
		fromColumns: fromColumns,
		toColumns:   toColumns,
		fromIndexes: fromIndexes,
		toIndexes:   toIndexes,
		from:        from,
		to:          to,
	}
}

func alterTables(ctx *diffCtx, dst io.Writer) (int64, error) {
	procs := []func(*alterCtx, io.Writer) (int64, error){
		dropTableColumns,
		addTableColumns,
		alterTableColumns,
		dropTableIndexes,
		addTableIndexes,
	}

	ids := ctx.toSet.Intersect(ctx.fromSet)
	var buf bytes.Buffer
	for _, id := range ids.ToSlice() {
		var stmt model.Stmt
		var ok bool

		stmt, ok = ctx.from.Lookup(id.(string))
		if !ok {
			return 0, errors.Errorf(`table '%s' not found in old schema (alter table)`, id)
		}
		beforeStmt := stmt.(model.Table)

		stmt, ok = ctx.to.Lookup(id.(string))
		if !ok {
			return 0, errors.Errorf(`table '%s' not found in new schema (alter table)`, id)
		}
		afterStmt := stmt.(model.Table)

		var pbuf bytes.Buffer
		alterCtx := newAlterCtx(beforeStmt, afterStmt)
		for _, p := range procs {
			n, err := p(alterCtx, &pbuf)
			if err != nil {
				return 0, errors.Wrap(err, `failed to generate alter table`)
			}

			if buf.Len() > 0 && n > 0 {
				buf.WriteByte('\n')
			}
			pbuf.WriteTo(&buf)
		}
	}

	return buf.WriteTo(dst)
}

func dropTableColumns(ctx *alterCtx, dst io.Writer) (int64, error) {
	columnNames := ctx.fromColumns.Difference(ctx.toColumns)

	var buf bytes.Buffer
	for _, columnName := range columnNames.ToSlice() {
		if buf.Len() > 0 {
			buf.WriteByte('\n')
		}
		buf.WriteString("ALTER TABLE `")
		buf.WriteString(ctx.from.Name())
		buf.WriteString("` DROP COLUMN `")
		col, ok := ctx.from.LookupColumn(columnName.(string))
		if !ok {
			return 0, errors.Errorf(`failed to lookup column %s`, columnName)
		}

		buf.WriteString(col.Name())
		buf.WriteString("`;")
	}

	return buf.WriteTo(dst)
}

func addTableColumns(ctx *alterCtx, dst io.Writer) (int64, error) {
	var buf bytes.Buffer

	columnNames := ctx.toColumns.Difference(ctx.fromColumns)
	for _, columnName := range columnNames.ToSlice() {
		stmt, ok := ctx.to.LookupColumn(columnName.(string))
		if !ok {
			continue
		}

		if buf.Len() > 0 {
			buf.WriteByte('\n')
		}
		buf.WriteString("ALTER TABLE `")
		buf.WriteString(ctx.from.Name())
		buf.WriteString("` ADD COLUMN ")
		if err := format.SQL(&buf, stmt); err != nil {
			return 0, err
		}
		buf.WriteByte(';')
	}

	return buf.WriteTo(dst)
}

func alterTableColumns(ctx *alterCtx, dst io.Writer) (int64, error) {
	var buf bytes.Buffer
	columnNames := ctx.toColumns.Intersect(ctx.fromColumns)
	for _, columnName := range columnNames.ToSlice() {
		beforeColumnStmt, ok := ctx.from.LookupColumn(columnName.(string))
		if !ok {
			return 0, errors.Errorf(`column %s not found in old schema`, columnName)
		}

		afterColumnStmt, ok := ctx.to.LookupColumn(columnName.(string))
		if !ok {
			return 0, errors.Errorf(`column %s not found in new schema`, columnName)
		}

		if reflect.DeepEqual(beforeColumnStmt, afterColumnStmt) {
			continue
		}

		if buf.Len() > 0 {
			buf.WriteByte('\n')
		}
		buf.WriteString("ALTER TABLE `")
		buf.WriteString(ctx.from.Name())
		buf.WriteString("` CHANGE COLUMN `")
		buf.WriteString(afterColumnStmt.Name())
		buf.WriteString("` ")
		if err := format.SQL(&buf, afterColumnStmt); err != nil {
			return 0, err
		}
		buf.WriteByte(';')
	}

	return buf.WriteTo(dst)
}

func dropTableIndexes(ctx *alterCtx, dst io.Writer) (int64, error) {
	var buf bytes.Buffer
	indexes := ctx.fromIndexes.Difference(ctx.toIndexes)
	for _, index := range indexes.ToSlice() {
		indexStmt, ok := ctx.from.LookupIndex(index.(string))
		if !ok {
			return 0, errors.Errorf(`index '%s' not found in old schema (drop index)`, index)
		}

		if indexStmt.IsPrimaryKey() {
			if buf.Len() > 0 {
				buf.WriteByte('\n')
			}
			buf.WriteString("ALTER TABLE `")
			buf.WriteString(ctx.from.Name())
			buf.WriteString("` DROP INDEX PRIMARY KEY;")
			continue
		}

		if !indexStmt.HasName() {
			return 0, errors.Errorf("can not drop index without name: %s", indexStmt.ID())
		}

		if buf.Len() > 0 {
			buf.WriteByte('\n')
		}
		buf.WriteString("ALTER TABLE `")
		buf.WriteString(ctx.from.Name())
		buf.WriteString("` DROP INDEX `")
		buf.WriteString(indexStmt.Name())
		buf.WriteString("`;")
	}

	return buf.WriteTo(dst)
}

func addTableIndexes(ctx *alterCtx, dst io.Writer) (int64, error) {
	var buf bytes.Buffer
	indexes := ctx.toIndexes.Difference(ctx.fromIndexes)
	for _, index := range indexes.ToSlice() {
		indexStmt, ok := ctx.to.LookupIndex(index.(string))
		if !ok {
			return 0, errors.Errorf(`index '%s' not found in old schema (add index)`, index)
		}
		if buf.Len() > 0 {
			buf.WriteByte('\n')
		}
		buf.WriteString("ALTER TABLE `")
		buf.WriteString(ctx.from.Name())
		buf.WriteString("` ADD ")
		if err := format.SQL(&buf, indexStmt); err != nil {
			return 0, err
		}
		buf.WriteByte(';')
	}

	return buf.WriteTo(dst)
}
