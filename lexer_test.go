package schemalex

import (
	"testing"
)

func TestRead(t *testing.T) {
	type Spec struct {
		input string
		str   string
		TokenType TokenType
	}

	specs := []Spec{
		// number
		{
			input: "123",
			str:   "123",
			TokenType: NUMBER,
		},
		{
			input: ".2",
			str:   ".2",
			TokenType: NUMBER,
		},
		{
			input: "3.4",
			str:   "3.4",
			TokenType: NUMBER,
		},
		{
			input: "-5",
			str:   "-5",
			TokenType: NUMBER,
		},
		{
			input: "-6.78",
			str:   "-6.78",
			TokenType: NUMBER,
		},
		{
			input: "+9.10",
			str:   "+9.10",
			TokenType: NUMBER,
		},
		{
			input: "1.2E3",
			str:   "1.2E3",
			TokenType: NUMBER,
		},
		{
			input: "1.2E-3",
			str:   "1.2E-3",
			TokenType: NUMBER,
		},
		{
			input: "-1.2E3",
			str:   "-1.2E3",
			TokenType: NUMBER,
		},
		{
			input: "-1.2E-3",
			str:   "-1.2E-3",
			TokenType: NUMBER,
		},
		// SINGLE_QUOTE_IDENT
		{
			input: `'hoge'`,
			str:   `'hoge'`,
			TokenType: SINGLE_QUOTE_IDENT,
		},
		{
			input: `'ho''ge'`,
			str:   `'ho'ge'`,
			TokenType: SINGLE_QUOTE_IDENT,
		},
		// DOUBLE_QUOTE_IDENT
		{
			input: `"hoge"`,
			str:   `"hoge"`,
			TokenType: DOUBLE_QUOTE_IDENT,
		},
		{
			input: `"ho""ge"`,
			str:   `"ho"ge"`,
			TokenType: DOUBLE_QUOTE_IDENT,
		},
		// BACKTICK_IDENT
		{
			input: "`hoge`",
			str:   "hoge",
			TokenType: BACKTICK_IDENT,
		},
		{
			input: "`ho``ge`",
			str:   "ho`ge",
			TokenType: BACKTICK_IDENT,
		},
		// ESCAPED STRING BY BACKSLASH
		{
			input: `'ho\'ge'`,
			str:   `'ho'ge'`,
			TokenType: SINGLE_QUOTE_IDENT,
		},
	}

	for _, spec := range specs {
		var l lexer
		l.input = []byte(spec.input)
		_t := l.read()
		if _t.Type != spec.TokenType {
			t.Errorf("got %d expected %d spec:%v", _t, spec.TokenType, spec)
		}
		if _t.Value != spec.str {
			t.Errorf("got %q expected %q spec:%v", _t.Value, spec.str, spec)
		}
	}
}
