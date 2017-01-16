package schemalex

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"golang.org/x/net/context"
)

func TestLexToken(t *testing.T) {
	type Spec struct {
		input string
		token Token
	}

	specs := []Spec{
		// number
		{
			input: "123",
			token: Token{Value: "123", Type: NUMBER},
		},
		{
			input: ".2",
			token: Token{Value: ".2", Type: NUMBER},
		},
		{
			input: "3.4",
			token: Token{Value: "3.4", Type: NUMBER},
		},
		{
			input: "-5",
			token: Token{Value: "-5", Type: NUMBER},
		},
		{
			input: "-6.78",
			token: Token{Value: "-6.78", Type: NUMBER},
		},
		{
			input: "+9.10",
			token: Token{Value: "+9.10", Type: NUMBER},
		},
		{
			input: "1.2E3",
			token: Token{Value: "1.2E3", Type: NUMBER},
		},
		{
			input: "1.2E-3",
			token: Token{Value: "1.2E-3", Type: NUMBER},
		},
		{
			input: "-1.2E3",
			token: Token{Value: "-1.2E3", Type: NUMBER},
		},
		{
			input: "-1.2E-3",
			token: Token{Value: "-1.2E-3", Type: NUMBER},
		},
		// SINGLE_QUOTE_IDENT
		{
			input: `'hoge'`,
			token: Token{Value: `hoge`, Type: SINGLE_QUOTE_IDENT},
		},
		{
			input: `'ho''ge'`,
			token: Token{Value: `ho'ge`, Type: SINGLE_QUOTE_IDENT},
		},
		// DOUBLE_QUOTE_IDENT
		{
			input: `"hoge"`,
			token: Token{Value: `hoge`, Type: DOUBLE_QUOTE_IDENT},
		},
		{
			input: `"ho""ge"`,
			token: Token{Value: `ho"ge`, Type: DOUBLE_QUOTE_IDENT},
		},
		// BACKTICK_IDENT
		{
			input: "`hoge`",
			token: Token{Value: "hoge", Type: BACKTICK_IDENT},
		},
		{
			input: "`ho``ge`",
			token: Token{Value: "ho`ge", Type: BACKTICK_IDENT},
		},
		// ESCAPED STRING BY BACKSLASH
		{
			input: `'ho\'ge'`,
			token: Token{Value: `ho'ge`, Type: SINGLE_QUOTE_IDENT},
		},
	}

	for _, spec := range specs {
		t.Logf("Lexing %s", spec.input)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		ch := lex(ctx, []byte(spec.input))
		select {
		case <-ctx.Done():
			t.Logf("%s", ctx.Err())
			t.Fail()
			return
		case tok := <-ch:
			spec.token.Line = 1
			spec.token.Col = 1
			if !assert.Equal(t, spec.token, *tok, "tok matches") {
				return
			}
		}
	}
}
