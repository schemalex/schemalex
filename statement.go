package schemalex

import "io"

type ider interface {
	ID() string
}

func (s Statements) WriteTo(dst io.Writer) (int64, error) {
	var n int64
	for _, stmt := range s {
		n1, err := stmt.WriteTo(dst)
		n += n1
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

// Lookup looks up statements by their ID, which could be their
// "name" or their stringified representation
func (s Statements) Lookup(id string) (Stmt, bool) {
	for _, stmt := range s {
		if n, ok := stmt.(ider); ok {
			if n.ID() == id {
				return stmt, true
			}
		}
	}
	return nil, false
}
