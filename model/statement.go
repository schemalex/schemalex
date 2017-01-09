package model

func (s Stmts) Lookup(id string) (Stmt, bool) {
	for _, stmt := range s {
		if stmt.ID() == id {
			return stmt, true
		}
	}
	return nil, false
}
