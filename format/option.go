package format

// Option is a generic interface for objects that passes
// optional parameters to the various format functions in this package
type Option interface {
	Name() string
	Value() interface{}
}

type option struct {
	name  string
	value interface{}
}

func (o option) Name() string       { return o.name }
func (o option) Value() interface{} { return o.value }

func WithIndent(s string) Option {
	return &option{
		name:  "indent",
		value: s,
	}
}
