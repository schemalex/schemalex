//go:generate go run internal/cmd/gentokens/main.go

package schemalex

// Backquote surrounds the given string in backquotes
func Backquote(s string) string {
	// XXX Does this require escaping
	return "`" + s + "`"
}
