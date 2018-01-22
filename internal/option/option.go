package option

type Option interface {
	Name() string
	Value() interface{}
}

type option struct {
	name  string
	value interface{}
}

func New(n string, v interface{}) Option {
	return &option{
		name:  n,
		value: v,
	}
}

func (o option) Name() string       { return o.name }
func (o option) Value() interface{} { return o.value }
