package diff

import (
	"github.com/eihigh/schemalex"
	"github.com/eihigh/schemalex/internal/option"
)

type Option = schemalex.Option

const (
	optkeyParser      = "parser"
	optkeyTransaction = "transaction"
)

// WithParser specifies the parser instance to use when parsing
// the statements given to the diffing functions. If unspecified,
// a default parser will be used
func WithParser(p *schemalex.Parser) Option {
	return option.New(optkeyParser, p)
}

// WithTransaction specifies if statements to control transactions
// should be included in the diff.
func WithTransaction(b bool) Option {
	return option.New(optkeyTransaction, b)
}
