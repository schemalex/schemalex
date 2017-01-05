package schemalex

import (
	"fmt"
	"io"

	"github.com/deckarep/golang-set"
)

type Differ struct {
	BeforeStmts []CreateTableStatement
	AfterStmts  []CreateTableStatement
}

func (d *Differ) WriteDiffWithTransaction(w io.Writer) error {
	stmts, err := d.DiffWithTransaction()
	if err != nil {
		return err
	}

	for _, stmt := range stmts {
		fmt.Fprintln(w, stmt+";\n")
	}

	return nil
}

func (d *Differ) DiffWithTransaction() ([]string, error) {
	var stmts []string

	diff, err := d.Diff()
	if err != nil {
		return nil, err
	}

	if len(diff) == 0 {
		return stmts, nil
	}

	stmts = append(stmts, "BEGIN")
	stmts = append(stmts, "SET FOREIGN_KEY_CHECKS = 0")
	stmts = append(stmts, diff...)
	stmts = append(stmts, "SET FOREIGN_KEY_CHECKS = 1")
	stmts = append(stmts, "COMMIT")

	return stmts, nil
}

func (d *Differ) Diff() ([]string, error) {
	var stmts []string

	// drop table
	stmts = append(stmts, d.dropTables()...)

	// create table
	stmts = append(stmts, d.creaeTables()...)

	// alter table
	{
		diffStmts, err := d.alterTables()
		if err != nil {
			return nil, err
		}
		stmts = append(stmts, diffStmts...)
	}

	return stmts, nil
}

func (d *Differ) dropTables() []string {
	var stmts []string

	before := d.beforeTableSets()
	after := d.afterTableSets()

	names := before.Difference(after)
	for _, name := range names.ToSlice() {
		name = name.(string)
		stmts = append(stmts, fmt.Sprintf("DROP TABLE `%s`", name))
	}

	return stmts
}

func (d *Differ) creaeTables() []string {
	var stmts []string

	before := d.beforeTableSets()
	after := d.afterTableSets()

	names := after.Difference(before)

	for _, name := range names.ToSlice() {
		for _, stmt := range d.AfterStmts {
			if stmt.Name == name.(string) {
				stmts = append(stmts, stmt.String())
				break
			}
		}
	}

	return stmts
}

func (d *Differ) alterTables() ([]string, error) {
	var stmts []string

	before := d.beforeTableSets()
	after := d.afterTableSets()

	names := after.Intersect(before)

	for _, name := range names.ToSlice() {
		var beforeStmt *CreateTableStatement
		var afterStmt *CreateTableStatement

		for _, stmt := range d.BeforeStmts {
			if stmt.Name == name.(string) {
				beforeStmt = &stmt
				break
			}
		}

		for _, stmt := range d.AfterStmts {
			if stmt.Name == name.(string) {
				afterStmt = &stmt
				break
			}
		}

		if beforeStmt == nil || afterStmt == nil {
			return nil, fmt.Errorf("table %s is not found before tables or after tables", name)
		}

		stmts = append(stmts, d.dropTableColumns(beforeStmt, afterStmt)...)
		stmts = append(stmts, d.addTableColumns(beforeStmt, afterStmt)...)
		{
			diffStmts, err := d.alterTableColumns(beforeStmt, afterStmt)
			if err != nil {
				return nil, err
			}
			stmts = append(stmts, diffStmts...)
		}

		{
			diffStmts, err := d.dropTableIndexes(beforeStmt, afterStmt)
			if err != nil {
				return nil, err
			}
			stmts = append(stmts, diffStmts...)
		}
		stmts = append(stmts, d.addTableIndexes(beforeStmt, afterStmt)...)
	}

	return stmts, nil
}

func (d *Differ) dropTableColumns(before *CreateTableStatement, after *CreateTableStatement) []string {
	var stmts []string

	beforeColumns := d.tableColumnNameSets(before)
	afterColumns := d.tableColumnNameSets(after)

	columnNames := beforeColumns.Difference(afterColumns)

	for _, columnName := range columnNames.ToSlice() {
		stmt := fmt.Sprintf("ALTER TABLE `%s` DROP COLUMN `%s`", before.Name, columnName.(string))
		stmts = append(stmts, stmt)
	}

	return stmts
}

func (d *Differ) addTableColumns(before *CreateTableStatement, after *CreateTableStatement) []string {
	var stmts []string

	beforeColumns := d.tableColumnNameSets(before)
	afterColumns := d.tableColumnNameSets(after)

	columnNames := afterColumns.Difference(beforeColumns)

	for _, columnName := range columnNames.ToSlice() {
		for _, columnStmt := range after.Columns {
			if columnStmt.Name == columnName {
				stmt := fmt.Sprintf("ALTER TABLE `%s` ADD COLUMN %s", before.Name, columnStmt.String())
				stmts = append(stmts, stmt)
				break
			}
		}
	}

	return stmts
}

func (d *Differ) alterTableColumns(before *CreateTableStatement, after *CreateTableStatement) ([]string, error) {
	var stmts []string

	beforeColumns := d.tableColumnNameSets(before)
	afterColumns := d.tableColumnNameSets(after)

	columnNames := afterColumns.Intersect(beforeColumns)

	for _, columnName := range columnNames.ToSlice() {
		var beforeColumnStmt *CreateTableColumnStatement
		var afterColumnStmt *CreateTableColumnStatement

		for _, columnStmt := range before.Columns {
			if columnStmt.Name == columnName {
				beforeColumnStmt = &columnStmt
				break
			}
		}

		for _, columnStmt := range after.Columns {
			if columnStmt.Name == columnName {
				afterColumnStmt = &columnStmt
				break
			}
		}

		if beforeColumnStmt == nil || afterColumnStmt == nil {
			return nil, fmt.Errorf("column %s is not found before columns or after columns", columnName)
		}

		if beforeColumnStmt.String() == afterColumnStmt.String() {
			continue
		}

		stmt := fmt.Sprintf("ALTER TABLE `%s` CHANGE COLUMN `%s` %s", before.Name, afterColumnStmt.Name, afterColumnStmt.String())
		stmts = append(stmts, stmt)
	}

	return stmts, nil
}

func (d *Differ) dropTableIndexes(before *CreateTableStatement, after *CreateTableStatement) ([]string, error) {
	var stmts []string

	beforeIndexes := d.tableIndexSets(before)
	afterIndexes := d.tableIndexSets(after)

	indexes := beforeIndexes.Difference(afterIndexes)

	for _, index := range indexes.ToSlice() {
		for _, indexStmt := range before.Indexes {
			if indexStmt.String() == index.(string) {
				var stmt string
				if indexStmt.Kind == IndexKindPrimaryKey {
					stmt = fmt.Sprintf("ALTER TABLE `%s` DROP INDEX PRIMARY KEY", before.Name)
				} else {
					if !indexStmt.Name.Valid {
						return nil, fmt.Errorf("cant drop index without name: %s", indexStmt.String())
					}
					stmt = fmt.Sprintf("ALTER TABLE `%s` DROP INDEX `%s`", before.Name, indexStmt.Name.Value)
				}
				stmts = append(stmts, stmt)
				break
			}
		}
	}

	return stmts, nil
}

func (d *Differ) addTableIndexes(before *CreateTableStatement, after *CreateTableStatement) []string {
	var stmts []string

	beforeIndexes := d.tableIndexSets(before)
	afterIndexes := d.tableIndexSets(after)

	indexes := afterIndexes.Difference(beforeIndexes)

	for _, index := range indexes.ToSlice() {
		stmt := fmt.Sprintf("ALTER TABLE `%s` ADD %s", before.Name, index)
		stmts = append(stmts, stmt)
	}

	return stmts
}

// return sets

func (d *Differ) beforeTableSets() mapset.Set {
	set := mapset.NewSet()
	for _, stmt := range d.BeforeStmts {
		set.Add(stmt.Name)
	}
	return set
}

func (d *Differ) afterTableSets() mapset.Set {
	set := mapset.NewSet()
	for _, stmt := range d.AfterStmts {
		set.Add(stmt.Name)
	}
	return set
}

func (d *Differ) tableColumnNameSets(stmt *CreateTableStatement) mapset.Set {
	set := mapset.NewSet()
	for _, col := range stmt.Columns {
		set.Add(col.Name)
	}
	return set
}

func (d *Differ) tableIndexSets(stmt *CreateTableStatement) mapset.Set {
	set := mapset.NewSet()
	for _, stmt := range stmt.Indexes {
		set.Add(stmt.String())
	}
	return set
}
