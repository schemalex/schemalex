// Package diff contains functions to generate SQL statements to
// migrate an old schema to the new schema
package diff

import (
	"bytes"
	"io"
	"reflect"
	"sort"

	"github.com/deckarep/golang-set"
	"github.com/schemalex/schemalex"
	"github.com/schemalex/schemalex/format"
	"github.com/schemalex/schemalex/internal/errors"
	"github.com/schemalex/schemalex/model"
)

type diffCtx struct {
	fromSet mapset.Set
	toSet   mapset.Set
	from    model.Stmts
	to      model.Stmts
	result  Stmts
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

// Diff compares two model.Stmts, and generates a series of
// statements as `diff.Stmts` so the consumer can, for example,
// analyze or use these statements standalone by themselves.
func Diff(from, to model.Stmts, options ...Option) (Stmts, error) {
	var txn bool
	for _, o := range options {
		switch o.Name() {
		case optkeyTransaction:
			txn = o.Value().(bool)
		}
	}

	ctx := newDiffCtx(from, to)

	var procs = []func(*diffCtx) error{
		dropTables,
		createTables,
		alterTables,
	}

	if txn {
		ctx.result.AppendStmt(`BEGIN`)
		ctx.result.AppendStmt(`SET FOREIGN_KEY_CHECKS = 0`)
	}

	for _, p := range procs {
		if err := p(ctx); err != nil {
			return nil, errors.Wrap(err, `failed to produce diff`)
		}
	}
	if txn {
		ctx.result.AppendStmt(`SET FOREIGN_KEY_CHECKS = 1`)
		ctx.result.AppendStmt(`COMMIT`)
	}
	return ctx.result, nil
}

// Statements compares two model.Stmts and generates a series
// of statements to migrate from the old one to the new one,
// writing the result to `dst`
func Statements(dst io.Writer, from, to model.Stmts, options ...Option) error {
	stmts, err := Diff(from, to, options...)
	if err != nil {
		return errors.Wrap(err, `failed to generate difference as statements`)
	}

	if _, err := stmts.WriteTo(dst); err != nil {
		return errors.Wrap(err, `failed to write diff`)
	}
	return nil
}

// Strings compares two strings and generates a series
// of statements to migrate from the old one to the new one,
// writing the result to `dst`
func Strings(dst io.Writer, from, to string, options ...Option) error {
	var p *schemalex.Parser
	for _, o := range options {
		switch o.Name() {
		case optkeyParser:
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

// Files compares contents of two files and generates a series
// of statements to migrate from the old one to the new one,
// writing the result to `dst`
func Files(dst io.Writer, from, to string, options ...Option) error {
	return Sources(dst, schemalex.NewLocalFileSource(from), schemalex.NewLocalFileSource(to), options...)
}

// Files compares contents from two sources and generates a series
// of statements to migrate from the old one to the new one,
// writing the result to `dst`
func Sources(dst io.Writer, from, to schemalex.SchemaSource, options ...Option) error {
	var buf bytes.Buffer
	if err := from.WriteSchema(&buf); err != nil {
		return errors.Wrapf(err, `failed to retrieve schema from "from" source %s`, from)
	}
	fromStr := buf.String()
	buf.Reset()

	if err := to.WriteSchema(&buf); err != nil {
		return errors.Wrapf(err, `failed to retrieve schema from "to" source %s`, to)
	}
	return Strings(dst, fromStr, buf.String(), options...)
}

func dropTables(ctx *diffCtx) error {
	ids := ctx.fromSet.Difference(ctx.toSet)
	for _, id := range ids.ToSlice() {
		stmt, ok := ctx.from.Lookup(id.(string))
		if !ok {
			return errors.Errorf(`failed to lookup table %s`, id)
		}

		table, ok := stmt.(model.Table)
		if !ok {
			return errors.Errorf(`lookup failed: %s is not a model.Table`, id)
		}
		ctx.result.AppendStmt("DROP TABLE `" + table.Name() + "`")
	}

	return nil
}

func createTables(ctx *diffCtx) error {
	ids := ctx.toSet.Difference(ctx.fromSet)
	var buf bytes.Buffer
	for _, id := range ids.ToSlice() {
		// Lookup the corresponding statement, and add its SQL
		stmt, ok := ctx.to.Lookup(id.(string))
		if !ok {
			return errors.Errorf(`failed to lookup table %s`, id)
		}

		buf.Reset()
		if err := format.SQL(&buf, stmt); err != nil {
			return errors.Wrap(err, `failed to format statement`)
		}
		ctx.result.AppendStmt(buf.String())
	}
	return nil
}

type alterCtx struct {
	fromColumns mapset.Set
	toColumns   mapset.Set
	fromIndexes mapset.Set
	toIndexes   mapset.Set
	from        model.Table
	to          model.Table
	result      Stmts
}

func newAlterCtx(ctx *diffCtx, from, to model.Table) *alterCtx {
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
		result:      ctx.result,
	}
}

func alterTables(ctx *diffCtx) error {
	procs := []func(*alterCtx) error{
		dropTableIndexes,
		dropTableColumns,
		addTableColumns,
		alterTableColumns,
		addTableIndexes,
	}

	ids := ctx.toSet.Intersect(ctx.fromSet)
	for _, id := range ids.ToSlice() {
		var stmt model.Stmt
		var ok bool

		stmt, ok = ctx.from.Lookup(id.(string))
		if !ok {
			return errors.Errorf(`table '%s' not found in old schema (alter table)`, id)
		}
		beforeStmt := stmt.(model.Table)

		stmt, ok = ctx.to.Lookup(id.(string))
		if !ok {
			return errors.Errorf(`table '%s' not found in new schema (alter table)`, id)
		}
		afterStmt := stmt.(model.Table)

		alterCtx := newAlterCtx(ctx, beforeStmt, afterStmt)
		for _, p := range procs {
			if err := p(alterCtx); err != nil {
				return errors.Wrap(err, `failed to generate alter table`)
			}
		}
		ctx.result = alterCtx.result
	}

	return nil
}

func dropTableColumns(ctx *alterCtx) error {
	columnNames := ctx.fromColumns.Difference(ctx.toColumns)

	for _, columnName := range columnNames.ToSlice() {
		col, ok := ctx.from.LookupColumn(columnName.(string))
		if !ok {
			return errors.Errorf(`failed to lookup column %s`, columnName)
		}

		ctx.result.AppendStmt("ALTER TABLE `" + ctx.from.Name() + "` DROP COLUMN `" + col.Name() + "`")
	}

	return nil
}

func addTableColumns(ctx *alterCtx, dst io.Writer) (int64, error) {
	var buf bytes.Buffer

	beforeToNext := make(map[string]string) // lookup next column
	nextToBefore := make(map[string]string) // lookup before column

	// In order to do this correctly, we need to create a graph so that
	// we always start adding with a column that has a either no before
	// columns, or one that already exists in the database
	var firstColumn model.TableColumn
	for _, v := range ctx.toColumns.Difference(ctx.fromColumns).ToSlice() {
		columnName := v.(string)
		// find the before-column for each.
		col, ok := ctx.to.LookupColumn(columnName)
		if !ok {
			return 0, errors.Errorf(`failed to lookup column %s`, columnName)
		}

		beforeCol, hasBeforeCol := ctx.to.LookupColumnBefore(col.ID())
		if !hasBeforeCol {
			// if there is no before-column, then this is a special "FIRST"
			// column
			firstColumn = col
			continue
		}

		// otherwise, keep a reverse-lookup map of before -> next columns
		beforeToNext[beforeCol.ID()] = columnName
		nextToBefore[columnName] = beforeCol.ID()
	}

	// First column is always safe to add
	if firstColumn != nil {
		writeAddColumn(ctx, &buf, firstColumn.ID())
	}

	var columnNames []string
	// Find columns that have before columns which existed in both
	// from and to tables
	for _, v := range ctx.toColumns.Intersect(ctx.fromColumns).ToSlice() {
		columnName := v.(string)
		if nextColumnName, ok := beforeToNext[columnName]; ok {
			delete(beforeToNext, columnName)
			delete(nextToBefore, nextColumnName)
			columnNames = append(columnNames, nextColumnName)
		}
	}

	if len(columnNames) > 0 {
		sort.Strings(columnNames)
		writeAddColumn(ctx, &buf, columnNames...)
	}

	// Finally, we process the remaining columns.
	// All remaining columns are new, and they will depend on a
	// newly created column. This means we have to make sure to
	// create them in the order that they are dependent on.
	columnNames = columnNames[:0]
	for _, nextCol := range beforeToNext {
		columnNames = append(columnNames, nextCol)
	}
	// if there's one left, that can be appended
	if len(columnNames) > 0 {
		sort.Slice(columnNames, func(i, j int) bool {
			icol, _ := ctx.to.LookupColumnOrder(columnNames[i])
			jcol, _ := ctx.to.LookupColumnOrder(columnNames[j])
			return icol < jcol
		})
		writeAddColumn(ctx, &buf, columnNames...)
	}
	return buf.WriteTo(dst)
}

func writeAddColumn(ctx *alterCtx, buf *bytes.Buffer, columnNames ...string) error {
	for _, columnName := range columnNames {
		stmt, ok := ctx.to.LookupColumn(columnName)
		if !ok {
			return errors.Errorf(`failed to lookup column %s`, columnName)
		}

		beforeCol, hasBeforeCol := ctx.to.LookupColumnBefore(stmt.ID())
		if buf.Len() > 0 {
			buf.WriteByte('\n')
		}
		buf.WriteString("ALTER TABLE `")
		buf.WriteString(ctx.from.Name())
		buf.WriteString("` ADD COLUMN ")
		if err := format.SQL(buf, stmt); err != nil {
			return err
		}
		if hasBeforeCol {
			buf.WriteString(" AFTER `")
			buf.WriteString(beforeCol.Name())
			buf.WriteString("`")
		} else {
			buf.WriteString(" FIRST")
		}

		buf.WriteByte(';')
	}
	return nil
}

func alterTableColumns(ctx *alterCtx) error {
	var buf bytes.Buffer
	columnNames := ctx.toColumns.Intersect(ctx.fromColumns)
	for _, columnName := range columnNames.ToSlice() {
		beforeColumnStmt, ok := ctx.from.LookupColumn(columnName.(string))
		if !ok {
			return errors.Errorf(`column %s not found in old schema`, columnName)
		}

		afterColumnStmt, ok := ctx.to.LookupColumn(columnName.(string))
		if !ok {
			return errors.Errorf(`column %s not found in new schema`, columnName)
		}

		if reflect.DeepEqual(beforeColumnStmt, afterColumnStmt) {
			continue
		}

		buf.Reset()
		buf.WriteString("ALTER TABLE `")
		buf.WriteString(ctx.from.Name())
		buf.WriteString("` CHANGE COLUMN `")
		buf.WriteString(afterColumnStmt.Name())
		buf.WriteString("` ")
		if err := format.SQL(&buf, afterColumnStmt); err != nil {
			return errors.Wrap(err, `failed to format statement`)
		}
		ctx.result.AppendStmt(buf.String())
	}

	return nil
}

func dropTableIndexes(ctx *alterCtx) error {
	var buf bytes.Buffer
	indexes := ctx.fromIndexes.Difference(ctx.toIndexes)
	// drop index after drop constraint.
	// because cannot drop index if needed in a foreign key constraint
	lazy := make([]model.Index, 0, indexes.Cardinality())
	for _, index := range indexes.ToSlice() {
		indexStmt, ok := ctx.from.LookupIndex(index.(string))
		if !ok {
			return errors.Errorf(`index '%s' not found in old schema (drop index)`, index)
		}

		if indexStmt.IsPrimaryKey() {
			ctx.result.AppendStmt("ALTER TABLE `" + ctx.from.Name() + "` DROP INDEX PRIMARY KEY")
			continue
		}

		if !indexStmt.HasName() && !indexStmt.HasSymbol() {
			return errors.Errorf("can not drop index without name: %s", indexStmt.ID())
		}

		if !indexStmt.IsForeginKey() {
			lazy = append(lazy, indexStmt)
			continue
		}

		buf.Reset()
		buf.WriteString("ALTER TABLE `")
		buf.WriteString(ctx.from.Name())
		buf.WriteString("` DROP FOREIGN KEY `")
		if indexStmt.HasSymbol() {
			buf.WriteString(indexStmt.Symbol())
		} else {
			buf.WriteString(indexStmt.Name())
		}
		buf.WriteString("`")
		ctx.result.AppendStmt(buf.String())
	}

	// drop index after drop CONSTRAINT
	for _, indexStmt := range lazy {
		buf.Reset()
		buf.WriteString("ALTER TABLE `")
		buf.WriteString(ctx.from.Name())
		buf.WriteString("` DROP INDEX `")
		if !indexStmt.HasName() {
			buf.WriteString(indexStmt.Symbol())
		} else {
			buf.WriteString(indexStmt.Name())
		}
		buf.WriteString("`")
		ctx.result.AppendStmt(buf.String())
	}

	return nil
}

func addTableIndexes(ctx *alterCtx) error {
	var buf bytes.Buffer
	indexes := ctx.toIndexes.Difference(ctx.fromIndexes)
	// add index before add foreign key.
	// because cannot add index if create implicitly index by foreign key.
	lazy := make([]model.Index, 0, indexes.Cardinality())
	for _, index := range indexes.ToSlice() {
		indexStmt, ok := ctx.to.LookupIndex(index.(string))
		if !ok {
			return errors.Errorf(`index '%s' not found in old schema (add index)`, index)
		}
		if indexStmt.IsForeginKey() {
			lazy = append(lazy, indexStmt)
			continue
		}

		buf.Reset()
		buf.WriteString("ALTER TABLE `")
		buf.WriteString(ctx.from.Name())
		buf.WriteString("` ADD ")
		if err := format.SQL(&buf, indexStmt); err != nil {
			return errors.Wrap(err, `failed to format statement`)
		}
		ctx.result.AppendStmt(buf.String())
	}

	for _, indexStmt := range lazy {
		buf.Reset()
		buf.WriteString("ALTER TABLE `")
		buf.WriteString(ctx.from.Name())
		buf.WriteString("` ADD ")
		if err := format.SQL(&buf, indexStmt); err != nil {
			return errors.Wrap(err, `failed to format statement`)
		}
		ctx.result.AppendStmt(buf.String())
	}

	return nil
}
