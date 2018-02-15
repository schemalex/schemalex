package diff

import (
	"io"

	"github.com/schemalex/schemalex/internal/errors"
)

func (s Stmt) String() string {
	return string(s)
}

func (stmts *Stmts) AppendStmt(s string) *Stmts {
	*stmts = append(*stmts, Stmt(s))
	return stmts
}

func (stmts *Stmts) WriteTo(dst io.Writer) (int64, error) {
	semicolon := []byte{';'}
	newline := []byte{'\n'}
	var sofar int64
	for i, s := range *stmts {
		if i > 0 {
			dst.Write(newline)
		}
		n, err := io.WriteString(dst, s.String())
		sofar += int64(n)
		if err != nil {
			return sofar, errors.Wrapf(err, `failed to write statement '%s'`, s.String())
		}
		dst.Write(semicolon)
	}
	return sofar, nil
}
