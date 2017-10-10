package model

// NewTable create a new table with the given name
func NewTable(name string) Table {
	return &table{
		name: name,
	}
}

func (t *table) ID() string {
	return "table#" + t.name
}

func (t *table) LookupColumn(name string) (TableColumn, bool) {
	for col := range t.Columns() {
		if col.ID() == name {
			return col, true
		}
	}
	return nil, false
}

func (t *table) LookupIndex(id string) (Index, bool) {
	for idx := range t.Indexes() {
		if idx.ID() == id {
			return idx, true
		}
	}
	return nil, false
}

func (t *table) AddColumn(v TableColumn) {
	t.columns = append(t.columns, v)
}

func (t *table) AddIndex(v Index) {
	t.indexes = append(t.indexes, v)
}

func (t *table) AddOption(v TableOption) {
	t.options = append(t.options, v)
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

func (t *table) SetIfNotExists(v bool) {
	t.ifnotexists = v
}

func (t *table) SetTemporary(v bool) {
	t.temporary = v
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

func (t *table) Normalize() Table {
	tbl := NewTable(t.Name())
	tbl.SetIfNotExists(t.IsIfNotExists())
	tbl.SetTemporary(t.IsTemporary())

	for col := range t.Columns() {
		ncol := col.Normalize()
		// column_definition [UNIQUE [KEY] | [PRIMARY] KEY]
		// they mean same as INDEX or CONSTRAINT
		switch {
		case ncol.IsPrimary():
			index := NewIndex(IndexKindPrimaryKey, tbl.ID())
			index.SetType(IndexTypeNone)
			index.AddColumns(ncol.Name())
			tbl.AddIndex(index)
			ncol.SetPrimary(false)
		case ncol.IsUnique():
			index := NewIndex(IndexKindUnique, tbl.ID())
			//  if you do not assign a name, the index is assigned the same name as the first indexed column
			index.SetName(ncol.Name())
			index.SetType(IndexTypeNone)
			index.AddColumns(ncol.Name())
			tbl.AddIndex(index)
			ncol.SetUnique(false)
		}
		tbl.AddColumn(ncol)
	}
	for idx := range t.Indexes() {
		nidx := idx.Normalize()
		switch {
		case nidx.IsForeginKey():
			// add implicitly created INDEX
			index := NewIndex(IndexKindNormal, tbl.ID())
			switch {
			case nidx.Symbol() != "":
				index.SetName(nidx.Symbol())
				index.SetType(IndexTypeNone)
				columns := []string{}
				for c := range nidx.Columns() {
					columns = append(columns, c)
				}
				index.AddColumns(columns...)
				tbl.AddIndex(index)
			default:
				// if Not defined CONSTRAINT symbol, then resolve implicitly created INDEX too difficult.
			}
		}
		tbl.AddIndex(nidx)
	}
	for opt := range t.Options() {
		tbl.AddOption(opt)
	}
	return tbl
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
