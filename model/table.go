package model

// NewTable create a new table with the given name
func NewTable(name string) Table {
	return &table{
		name:              name,
		columnNameToIndex: make(map[string]int),
	}
}

func (t *table) ID() string {
	return "table#" + t.name
}

func (t *table) lookupColumnOrderNoLock(id string) (int, bool) {
	idx, ok := t.columnNameToIndex[id]
	return idx, ok
}

func (t *table) LookupColumn(id string) (TableColumn, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	idx, ok := t.lookupColumnOrderNoLock(id)
	if !ok {
		return nil, false
	}

	return t.columns[idx], true
}

func (t *table) LookupColumnOrder(id string) (int, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	idx, ok := t.lookupColumnOrderNoLock(id)
	return idx, ok
}

func (t *table) LookupColumnBefore(id string) (TableColumn, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	idx, ok := t.columnNameToIndex[id]
	if !ok || idx == 0 {
		return nil, false
	}

	return t.columns[idx-1], true
}

func (t *table) LookupIndex(id string) (Index, bool) {
	for idx := range t.Indexes() {
		if idx.ID() == id {
			return idx, true
		}
	}
	return nil, false
}

func (t *table) AddColumn(v TableColumn) Table {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Avoid adding the same TableColumn to multiple tables
	if tblID := v.TableID(); tblID != "" {
		v = v.Clone()
	}
	v.SetTableID(t.ID())
	t.columns = append(t.columns, v)
	t.columnNameToIndex[v.ID()] = len(t.columns) - 1
	return t
}

func (t *table) AddIndex(v Index) Table {
	t.indexes = append(t.indexes, v)
	return t
}

func (t *table) AddOption(v TableOption) Table {
	t.options = append(t.options, v)
	return t
}

func (t *table) Name() string {
	return t.name
}

func (t *table) IsIfNotExists() bool {
	return t.ifnotexists
}

func (t *table) IsTemporary() bool {
	return t.temporary
}

func (t *table) SetIfNotExists(v bool) Table {
	t.ifnotexists = v
	return t
}

func (t *table) HasLikeTable() bool {
	return t.likeTable.Valid
}

func (t *table) SetLikeTable(s string) Table {
	t.likeTable.Valid = true
	t.likeTable.Value = s
	return t
}

func (t *table) LikeTable() string {
	return t.likeTable.Value
}

func (t *table) SetTemporary(v bool) Table {
	t.temporary = v
	return t
}

func (t *table) Columns() chan TableColumn {
	ch := make(chan TableColumn, len(t.columns))
	for _, col := range t.columns {
		ch <- col
	}
	close(ch)
	return ch
}

func (t *table) Indexes() chan Index {
	ch := make(chan Index, len(t.indexes))
	for _, idx := range t.indexes {
		ch <- idx
	}
	close(ch)
	return ch
}

func (t *table) Options() chan TableOption {
	ch := make(chan TableOption, len(t.options))
	for _, idx := range t.options {
		ch <- idx
	}
	close(ch)
	return ch
}

func (t *table) Normalize() (Table, bool) {
	var clone bool
	var additionalIndexes []Index
	var columns []TableColumn
	for col := range t.Columns() {
		ncol, modified := col.Normalize()
		if modified {
			clone = true
		}

		// column_definition [UNIQUE [KEY] | [PRIMARY] KEY]
		// they mean same as INDEX or CONSTRAINT
		switch {
		case ncol.IsPrimary():
			// we have to move off the index declaration from the
			// primary key column to an index associated with the table
			index := NewIndex(IndexKindPrimaryKey, t.ID())
			index.SetType(IndexTypeNone)
			idxCol := NewIndexColumn(ncol.Name())
			index.AddColumns(idxCol)
			additionalIndexes = append(additionalIndexes, index)
			if !modified {
				clone = true
			}
			ncol = ncol.Clone()
			ncol.SetPrimary(false)
		case ncol.IsUnique():
			index := NewIndex(IndexKindUnique, t.ID())
			// if you do not assign a name, the index is assigned the same name as the first indexed column
			index.SetName(ncol.Name())
			index.SetType(IndexTypeNone)
			idxCol := NewIndexColumn(ncol.Name())
			index.AddColumns(idxCol)
			additionalIndexes = append(additionalIndexes, index)
			if !modified {
				clone = true
			}
			ncol = ncol.Clone()
			ncol.SetUnique(false)
		}

		columns = append(columns, ncol)
	}

	var indexes []Index
	var seen = make(map[string]struct{})
	for idx := range t.Indexes() {
		nidx, modified := idx.Normalize()
		if modified {
			clone = true
		}

		// if Not defined CONSTRAINT symbol, then resolve
		// implicitly created INDEX too difficult.
		// (lestrrat) this comment is confusing. Please add
		// actual examples somewhere
		if nidx.IsForeignKey() && nidx.HasSymbol() {
			// There's a chance the user has already explicitly declared the
			// index for this constraint. Only add this implicit index if we
			// haven't seen it before
			if _, ok := seen[nidx.Symbol()]; !ok {
				clone = true
				// add implicitly created INDEX
				index := NewIndex(IndexKindNormal, t.ID())
				index.SetName(nidx.Symbol())
				if nidx.IsBtree() {
					index.SetType(IndexTypeBtree)
				} else if nidx.IsHash() {
					index.SetType(IndexTypeHash)
				}
				columns := []IndexColumn{}
				for c := range nidx.Columns() {
					columns = append(columns, c)
				}
				index.AddColumns(columns...)
				indexes = append(indexes, index)
			}
		}
		indexes = append(indexes, nidx)
		seen[nidx.Name()] = struct{}{}
	}

	if !clone {
		return t, false
	}

	tbl := NewTable(t.Name())
	tbl.SetIfNotExists(t.IsIfNotExists())
	tbl.SetTemporary(t.IsTemporary())

	for _, index := range additionalIndexes {
		tbl.AddIndex(index)
	}

	for _, col := range columns {
		tbl.AddColumn(col)
	}

	for _, idx := range indexes {
		tbl.AddIndex(idx)
	}

	for opt := range t.Options() {
		tbl.AddOption(opt)
	}
	return tbl, true
}

// NewTableOption creates a new table option with the given name, value, and a flag indicating if quoting is necessary
func NewTableOption(k, v string, q bool) TableOption {
	return &tableopt{
		key:        k,
		value:      v,
		needQuotes: q,
	}
}

func (t *tableopt) ID() string       { return "tableopt#" + t.key }
func (t *tableopt) Key() string      { return t.key }
func (t *tableopt) Value() string    { return t.value }
func (t *tableopt) NeedQuotes() bool { return t.needQuotes }
