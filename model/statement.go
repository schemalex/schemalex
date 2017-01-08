package model

import "io"

func (s Stmts) WriteTo(dst io.Writer) (int64, error) {
	var total int64
	for _, stmt := range s {
		n, err := stmt.WriteTo(dst)
		total += n
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

func (s Stmts) Lookup(id string) (Stmt, bool) {
	for _, stmt := range s {
		if stmt.ID() == id {
			return stmt, true
		}
	}
	return nil, false
}
