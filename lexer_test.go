package schemalex

import (
	"testing"
)

func TestRead(t *testing.T) {
	type Spec struct {
		input string
		str   string
		token token
	}

	specs := []Spec{
		// number
		{
			input: "123",
			str:   "123",
			token: NUMBER,
		},
		{
			input: ".2",
			str:   ".2",
			token: NUMBER,
		},
		{
			input: "3.4",
			str:   "3.4",
			token: NUMBER,
		},
		{
			input: "-5",
			str:   "-5",
			token: NUMBER,
		},
		{
			input: "-6.78",
			str:   "-6.78",
			token: NUMBER,
		},
		{
			input: "+9.10",
			str:   "+9.10",
			token: NUMBER,
		},
		{
			input: "1.2E3",
			str:   "1.2E3",
			token: NUMBER,
		},
		{
			input: "1.2E-3",
			str:   "1.2E-3",
			token: NUMBER,
		},
		{
			input: "-1.2E3",
			str:   "-1.2E3",
			token: NUMBER,
		},
		{
			input: "-1.2E-3",
			str:   "-1.2E-3",
			token: NUMBER,
		},
		// SINGLE_QUOTE_IDENT
		{
			input: `'hoge'`,
			str:   `'hoge'`,
			token: SINGLE_QUOTE_IDENT,
		},
		{
			input: `'ho''ge'`,
			str:   `'ho'ge'`,
			token: SINGLE_QUOTE_IDENT,
		},
		// DOUBLE_QUOTE_IDENT
		{
			input: `"hoge"`,
			str:   `"hoge"`,
			token: DOUBLE_QUOTE_IDENT,
		},
		{
			input: `"ho""ge"`,
			str:   `"ho"ge"`,
			token: DOUBLE_QUOTE_IDENT,
		},
		// BACKTICK_IDENT
		{
			input: "`hoge`",
			str:   "hoge",
			token: BACKTICK_IDENT,
		},
		{
			input: "`ho``ge`",
			str:   "ho`ge",
			token: BACKTICK_IDENT,
		},
		// ESCAPED STRING BY BACKSLASH
		{
			input: `'ho\'ge'`,
			str:   `'ho'ge'`,
			token: SINGLE_QUOTE_IDENT,
		},
	}

	for _, spec := range specs {
		l := lexer{
			input: spec.input,
		}
		_t, s := l.read()
		if _t != spec.token {
			t.Errorf("got %d expected %d spec:%v", _t, spec.token, spec)
		}
		if s != spec.str {
			t.Errorf("got %q expected %q spec:%v", s, spec.str, spec)
		}
	}
}
