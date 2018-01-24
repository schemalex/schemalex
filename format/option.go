package format

import (
	"strings"

	"github.com/schemalex/schemalex"
	"github.com/schemalex/schemalex/internal/option"
)

type Option = schemalex.Option

const optkeyIndent = "indent"

// WithIndent specifies the indent string to use, and the length.
// For example, if you specify WithIndent(" " /* single space */, 2), the
// indent will be 2 spaces per level.
//
// Please note that no check on the string will be performed, so anything
// you specify will be used as-is.
func WithIndent(s string, n int) Option {
	if n <= 0 {
		n = 1
	}
	return option.New(optkeyIndent, strings.Repeat(s, n))
}
