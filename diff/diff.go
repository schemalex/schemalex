package diff

import (
	"bytes"
	"io"
	"reflect"

	"github.com/deckarep/golang-set"
	"github.com/lestrrat/schemalex"
	"github.com/pkg/errors"
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
	from    schemalex.Statements
	to      schemalex.Statements
}

func newDiffCtx(from, to schemalex.Statements) *diffCtx {
	fromSet := mapset.NewSet()
	for _, stmt := range from {
		if cs, ok := stmt.(*schemalex.CreateTableStatement); ok {
			fromSet.Add(cs.Name)
		}
	}
	toSet := mapset.NewSet()
	for _, stmt := range to {
		if cs, ok := stmt.(*schemalex.CreateTableStatement); ok {
			toSet.Add(cs.Name)
		}
	}

	return &diffCtx{
		fromSet: fromSet,
		toSet:   toSet,
		from:    from,
		to:      to,
	}
}

func Statements(dst io.Writer, from, to schemalex.Statements, options ...Option) error {
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
		if n > 0 {
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
	names := ctx.fromSet.Difference(ctx.toSet)
	for i, name := range names.ToSlice() {
		if i > 0 {
			buf.WriteByte('\n')
		}
		buf.WriteString("DROP TABLE `")
		buf.WriteString(name.(string))
		buf.WriteString("`;")
	}

	return buf.WriteTo(dst)
}

func createTables(ctx *diffCtx, dst io.Writer) (int64, error) {
	var buf bytes.Buffer

	names := ctx.toSet.Difference(ctx.fromSet)
	for _, name := range names.ToSlice() {
		// Lookup the corresponding statement, and add its SQL
		stmt, ok := ctx.to.Lookup(name.(string))
		if !ok {
			continue
		}

		if buf.Len() > 0 {
			buf.WriteByte('\n')
		}
		_, err := stmt.WriteTo(&buf)
		if err != nil {
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
	from        *schemalex.CreateTableStatement
	to          *schemalex.CreateTableStatement
}

func newAlterCtx(from, to *schemalex.CreateTableStatement) *alterCtx {
	fromColumns := mapset.NewSet()
	for _, col := range from.Columns {
		fromColumns.Add(col.Name)
	}

	toColumns := mapset.NewSet()
	for _, col := range to.Columns {
		toColumns.Add(col.Name)
	}

	fromIndexes := mapset.NewSet()
	for _, idx := range from.Indexes {
		fromIndexes.Add(idx.String())
	}

	toIndexes := mapset.NewSet()
	for _, idx := range to.Indexes {
		toIndexes.Add(idx.String())
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

	names := ctx.toSet.Intersect(ctx.fromSet)
	var buf bytes.Buffer
	for _, name := range names.ToSlice() {
		var stmt schemalex.Stmt
		var ok bool

		stmt, ok = ctx.from.Lookup(name.(string))
		if !ok {
			return 0, errors.Errorf(`table '%s' not found in old schema (alter table)`, name)
		}
		beforeStmt := stmt.(*schemalex.CreateTableStatement)

		stmt, ok = ctx.to.Lookup(name.(string))
		if !ok {
			return 0, errors.Errorf(`table '%s' not found in new schema (alter table)`, name)
		}
		afterStmt := stmt.(*schemalex.CreateTableStatement)

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
		buf.WriteString(ctx.from.Name)
		buf.WriteString("` DROP COLUMN `")
		buf.WriteString(columnName.(string))
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
		buf.WriteString(ctx.from.Name)
		buf.WriteString("` ADD COLUMN ")
		if _, err := stmt.WriteTo(&buf); err != nil {
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
		buf.WriteString(ctx.from.Name)
		buf.WriteString("` CHANGE COLUMN `")
		buf.WriteString(afterColumnStmt.Name)
		buf.WriteString("` ")
		if _, err := afterColumnStmt.WriteTo(&buf); err != nil {
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

		if indexStmt.Kind == schemalex.IndexKindPrimaryKey {
			if buf.Len() > 0 {
				buf.WriteByte('\n')
			}
			buf.WriteString("ALTER TABLE `")
			buf.WriteString(ctx.from.Name)
			buf.WriteString("` DROP INDEX PRIMARY KEY;")
			continue
		}

		if !indexStmt.Name.Valid {
			return 0, errors.Errorf("can not drop index without name: %s", indexStmt.String())
		}

		if buf.Len() > 0 {
			buf.WriteByte('\n')
		}
		buf.WriteString("ALTER TABLE `")
		buf.WriteString(ctx.from.Name)
		buf.WriteString("` DROP INDEX `")
		buf.WriteString(indexStmt.Name.Value)
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
		buf.WriteString(ctx.from.Name)
		buf.WriteString("` ADD ")
		buf.WriteString(indexStmt.String())
		buf.WriteByte(';')
	}

	return buf.WriteTo(dst)
}
