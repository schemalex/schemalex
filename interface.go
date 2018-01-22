package schemalex

// Option is a generic interface for objects that passes
// optional parameters to the various format functions in this package
type Option interface {
	Name() string
	Value() interface {}
}