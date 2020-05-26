package util

import (
	"strings"
)

// Backquote surrounds the given string in backquotes
func Backquote(s string) string {
	// XXX Does this require escaping
	return "`" + s + "`"
}

// Singlequote surrounds the given string in singlequotes
func Singlequote(s string) string {
	b := strings.Builder{}
	b.Grow(len(s) + 2)
	b.WriteRune('\'')
	r := strings.NewReader(s)
	for {
		c, _, err := r.ReadRune()
		if err != nil {
			break
		}
		switch c {
		case '\'', '\\':
			b.WriteRune('\\')
		}
		b.WriteRune(c)
	}
	b.WriteRune('\'')
	return b.String()
}
