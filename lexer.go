package schemalex

import (
	"bytes"
	"strings"
	"unicode/utf8"

	"golang.org/x/net/context"
)

const eof = rune(0)

type lrune struct {
	r rune
	w int
}

type lexer struct {
	out       chan *Token
	input     []byte
	peekCount int
	peekRunes [3]lrune

	pos   int
	start int
	width int
	line  int
	col   int
}

func Lex(ctx context.Context, input []byte) chan *Token {
	ch := make(chan *Token, 3)
	l := newLexer(ch, input)
	go l.Run(ctx)
	return ch
}

func newLexer(out chan *Token, input []byte) *lexer {
	return &lexer{
		out:       out,
		input:     input,
		line:      1,
		col:       1,
		peekCount: -1,
	}
}

func (l *lexer) emit(ctx context.Context, t *Token) {
	//t.Pos = l.start // TODO check if this is correct
	select {
	case <-ctx.Done():
		return
	case l.out <- t:
		return
	}
}

func (l *lexer) str() string {
	return string(l.input[l.start:l.pos-(l.peekCount+1)])
}

func (l *lexer) Run(ctx context.Context) {
	defer close(l.out)

OUTER:
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		l.start = l.pos-(l.peekCount+1)

		r := l.peek()

		// These require peek, and then consume
		switch {
		case isSpace(r):
			// read until space end
			l.runSpace()
			l.emit(ctx, &Token{Type: SPACE})
			continue OUTER
		case isLetter(r):
			t, s := l.runIdent(), l.str()
			if _t, ok := keywordIdentMap[strings.ToUpper(s)]; ok {
				t = _t
			}
			l.emit(ctx, &Token{Type: t, Value: s})
			continue OUTER
		case isDigit(r):
			l.emit(ctx, l.runNumber())
			continue OUTER
		}

		// once we got here, we can consume
		l.advance()
		switch r {
		case eof:
			l.emit(ctx, &Token{Type: EOF})
		case '`':
			l.emit(ctx, l.runQuote('`', BACKTICK_IDENT))
		case '"':
			t := l.runQuote('"', DOUBLE_QUOTE_IDENT)
			if t.Type == DOUBLE_QUOTE_IDENT {
				t.Value = `"` + t.Value + `"`
			}
			l.emit(ctx, t)
		case '\'':
			t := l.runQuote('\'', SINGLE_QUOTE_IDENT)
			if t.Type == SINGLE_QUOTE_IDENT {
				t.Value = `'` + t.Value + `'`
			}
			l.emit(ctx, t)
		case '/':
			switch c := l.peek(); c {
			case '*':
				l.emit(ctx, &Token{Type: l.runCComment()})
			default:
				l.emit(ctx, &Token{Type: SLASH})
			}
		case '-':
			switch r1 := l.peek(); {
			case r1 == '-':
				l.advance()
				// TODO: https://dev.mysql.com/doc/refman/5.6/en/comments.html
				// TODO: not only space. control character
				if !isSpace(l.peek()) {
					l.emit(ctx, &Token{Type: DASH})
					continue OUTER
				}
				l.runToEOL()
				l.emit(ctx, &Token{Type: COMMENT_IDENT})
			case isDigit(r1):
				l.emit(ctx, l.runNumber())
			default:
				l.emit(ctx, &Token{Type: DASH})
			}
		case '#':
			// https://dev.mysql.com/doc/refman/5.6/en/comments.html
			l.runToEOL()
			l.emit(ctx, &Token{Type: COMMENT_IDENT})
		case '(':
			l.emit(ctx, &Token{Type: LPAREN})
		case ')':
			l.emit(ctx, &Token{Type: RPAREN})
		case ';':
			l.emit(ctx, &Token{Type: SEMICOLON})
		case ',':
			l.emit(ctx, &Token{Type: COMMA})
		case '.':
			if isDigit(l.peek()) {
				l.emit(ctx, l.runNumber())
			} else {
				l.emit(ctx, &Token{Type: DOT})
			}
		case '+':
			if isDigit(l.peek()) {
				l.emit(ctx, l.runNumber())
			} else {
				l.emit(ctx, &Token{Type: PLUS})
			}
		case '=':
			l.emit(ctx, &Token{Type: EQUAL})
		default:
			l.emit(ctx, &Token{Type: ILLEGAL})
		}
	}
}

func (l *lexer) next() rune {
	r := l.peek()
	l.advance()
	return r
}

func (l *lexer) peek() rune {
	if l.peekCount >= 0 {
		return l.peekRunes[l.peekCount].r
	}

	if l.pos >= len(l.input) {
		l.width = 0
		return eof
	}

	r, w := utf8.DecodeRune(l.input[l.pos:])
	l.peekCount++
	l.peekRunes[l.peekCount].r = r
	l.peekRunes[l.peekCount].w = w
	l.pos += w

	return r
}

func (l *lexer) advance() {
	l.peekCount--
}

func (l *lexer) runSpace() {
	for isSpace(l.peek()) {
		l.advance()
	}
}

func (l *lexer) runIdent() TokenType {
OUTER:
	for {
		r := l.peek()
		switch {
		case r == eof:
			l.advance()
			break OUTER
		case isCharacter(r):
			l.advance()
		default:
			break OUTER
		}
	}
	return IDENT
}

func (l *lexer) runQuote(pair rune, t TokenType) *Token {
	var b bytes.Buffer
	for {
		r := l.next()
		if r == eof {
			return &Token{Type: ILLEGAL}
		} else if r == '\\' {
			if l.peek() == pair {
				r = l.next()
			}
		} else if r == pair {
			if l.peek() == pair {
				// it is escape
				r = l.next()
			} else {
				return &Token{Type: t, Value: b.String()}
			}
		}
		b.WriteRune(r)
	}

	return &Token{Type: ILLEGAL}
}

// https://dev.mysql.com/doc/refman/5.6/en/comments.html
func (l *lexer) runCComment() TokenType {
	for {
		r := l.next()
		switch r {
		case eof:
			return EOF
		case '*':
			if l.peek() == '/' {
				l.advance()
				return COMMENT_IDENT
			}
		}
	}
}

func (l *lexer) runToEOL() TokenType {
	for {
		r := l.next()
		switch r {
		case eof, '\n':
			return COMMENT_IDENT
		}
	}
}

// https://dev.mysql.com/doc/refman/5.6/en/number-literals.html
func (l *lexer) runDigit() {
	for {
		if !isDigit(l.peek()) {
			break
		}
		l.advance()
	}
}

func (l *lexer) runNumber() *Token {
	l.runDigit()
	if l.peek() == '.' {
		l.advance()
		l.runDigit()
	}

	switch l.peek() {
	case 'E', 'e':
		l.advance()
		if l.peek() == '-' {
			l.advance()
		}
		l.runDigit()
	}

	return &Token{Type: NUMBER, Value: l.str()}
}

func isSpace(r rune) bool {
	return r == ' ' || r == '\n' || r == '\t'
}

func isLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func isCharacter(r rune) bool {
	return isDigit(r) || isLetter(r) || r == '_'
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}
