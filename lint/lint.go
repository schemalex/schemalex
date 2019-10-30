package lint

import (
	"bytes"
	"context"
	"io"

	"github.com/pkg/errors"
	"github.com/eihigh/schemalex"
	"github.com/eihigh/schemalex/format"
)

type Linter struct{}
type Option = schemalex.Option

func WithIndent(s string, n int) Option {
	return format.WithIndent(s, n)
}

func New(options ...Option) *Linter {
	return &Linter{}
}

func (l *Linter) Run(ctx context.Context, src schemalex.SchemaSource, dst io.Writer, options ...Option) error {
	var buf bytes.Buffer
	if err := src.WriteSchema(&buf); err != nil {
		return errors.Wrap(err, `failed to read from source`)
	}

	p := schemalex.New()
	stmts, err := p.Parse(buf.Bytes())
	if err != nil {
		return errors.Wrap(err, `failed to parse source`)
	}

	for i, stmt := range stmts {
		if i != 0 {
			dst.Write([]byte{'\n', '\n'})
		}

		if err := format.SQL(dst, stmt, options...); err != nil {
			return errors.Wrap(err, `failed to format source`)
		}
		dst.Write([]byte{';'})
	}

	return nil
}
