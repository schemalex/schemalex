package diff

import (
	"github.com/schemalex/schemalex"
	"github.com/schemalex/schemalex/internal/option"
)

type Option = schemalex.Option

const (
	optKeyParser      = "parser"
	optKeyTransaction = "transaction"
)

// WithParser specifies the parser instance to use when parsing
// the statements given to the diffing functions. If unspecified,
// a default parser will be used
func WithParser(p *schemalex.Parser) Option {
	return option.New(optKeyParser, p)
}

// WithTransaction specifies if statements to control transactions
// should be included in the diff.
func WithTransaction(b bool) Option {
	return option.New(optKeyTransaction, b)
}