package schemalex

import (
	"bytes"
	"context"
	"strings"
	"unicode/utf8"

	"github.com/schemalex/schemalex/internal/errors"
)

const eof = rune(0)

type lrune struct {
	r rune
	w int
}

type position struct {
	pos  int // byte count
	col  int
	line int
}

type lexer struct {
	out       chan *Token
	input     []byte
	peekCount int
	peekRunes [3]lrune

	start position // position where we last emitted
	cur   position // current position including read-ahead
	width int
}

func lex(ctx context.Context, input []byte) chan *Token {
	ch := make(chan *Token, 3)
	l := newLexer(ch, input)
	go l.Run(ctx)
	return ch
}

func newLexer(out chan *Token, input []byte) *lexer {
	var l lexer
	l.out = out
	l.input = input
	l.start.line = 1
	l.start.col = 1
	l.cur.line = 1
	l.cur.col = 1
	l.peekCount = -1
	return &l
}

func (l *lexer) emit(ctx context.Context, typ TokenType) {
	var t Token
	t.Line = l.start.line
	t.Col = l.start.col
	t.Type = typ
	t.Pos = l.start.pos

	if typ == EOF {
		t.EOF = true
		t.Pos = len(l.input)
	} else {
		t.Value = l.str()
		switch typ {
		case SINGLE_QUOTE_IDENT:
			t.Value = unescapeQuotes(t.Value, '\'')
		case DOUBLE_QUOTE_IDENT:
			t.Value = unescapeQuotes(t.Value, '"')
		case BACKTICK_IDENT:
			t.Value = unescapeQuotes(t.Value, '`')
		}
	}

	select {
	case <-ctx.Done():
	case l.out <- &t:
	}

	// when we emit, we must copy the value of cur to start
	// but we also must adjust the position by the read-ahead offset
	l.start = l.cur
	l.start.pos = l.start.pos - (l.peekCount + 1)
}

func (l *lexer) str() string {
	endpos := l.cur.pos - (l.peekCount + 1)
	w := len(l.input[l.start.pos:])
	if endpos-l.start.pos > w {
		endpos = l.start.pos + w
	}
	return string(l.input[l.start.pos:endpos])
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

		r := l.peek()

		// These require peek, and then consume
		switch {
		case isSpace(r):
			// read until space end
			l.runSpace()
			l.emit(ctx, SPACE)
			continue OUTER
		case isLetter(r):
			t := l.runIdent()
			s := l.str()
			if typ, ok := keywordIdentMap[strings.ToUpper(s)]; ok {
				t = typ
			}
			l.emit(ctx, t)
			continue OUTER
		case isDigit(r):
			l.runNumber()
			l.emit(ctx, NUMBER)
			continue OUTER
		}

		// once we got here, we can consume
		l.advance()
		switch r {
		case eof:
			l.emit(ctx, EOF)
			return
		case '`':
			if err := l.runQuote('`'); err != nil {
				l.emit(ctx, ILLEGAL)
				return
			}

			l.emit(ctx, BACKTICK_IDENT)
		case '"':
			if err := l.runQuote('"'); err != nil {
				l.emit(ctx, ILLEGAL)
				return
			}

			l.emit(ctx, DOUBLE_QUOTE_IDENT)
		case '\'':
			if err := l.runQuote('\''); err != nil {
				l.emit(ctx, ILLEGAL)
				return
			}

			l.emit(ctx, SINGLE_QUOTE_IDENT)
		case '/':
			switch c := l.peek(); c {
			case '*':
				l.runCComment()
				l.emit(ctx, COMMENT_IDENT)
			default:
				l.emit(ctx, SLASH)
			}
		case '-':
			switch r1 := l.peek(); {
			case r1 == '-':
				l.advance()
				// TODO: https://dev.mysql.com/doc/refman/5.6/en/comments.html
				// TODO: not only space. control character
				if !isSpace(l.peek()) {
					l.emit(ctx, DASH)
					continue OUTER
				}
				l.runToEOL()
				l.emit(ctx, COMMENT_IDENT)
			case isDigit(r1):
				l.runNumber()
				l.emit(ctx, NUMBER)
			default:
				l.emit(ctx, DASH)
			}
		case '#':
			// https://dev.mysql.com/doc/refman/5.6/en/comments.html
			l.runToEOL()
			l.emit(ctx, COMMENT_IDENT)
		case '(':
			l.emit(ctx, LPAREN)
		case ')':
			l.emit(ctx, RPAREN)
		case ';':
			l.emit(ctx, SEMICOLON)
		case ',':
			l.emit(ctx, COMMA)
		case '.':
			if isDigit(l.peek()) {
				l.runNumber()
				l.emit(ctx, NUMBER)
			} else {
				l.emit(ctx, DOT)
			}
		case '+':
			if isDigit(l.peek()) {
				l.runNumber()
				l.emit(ctx, NUMBER)
			} else {
				l.emit(ctx, PLUS)
			}
		case '=':
			l.emit(ctx, EQUAL)
		default:
			l.emit(ctx, ILLEGAL)
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

	if l.cur.pos >= len(l.input) {
		l.width = 0
		return eof
	}

	r, w := utf8.DecodeRune(l.input[l.cur.pos:])
	l.peekCount++
	l.peekRunes[l.peekCount].r = r
	l.peekRunes[l.peekCount].w = w
	l.cur.pos += w

	return r
}

func (l *lexer) advance() {
	// if the current rune is a new line, we line++
	r := l.peek()
	switch r {
	case '\n':
		l.cur.line++
		l.cur.col = 0
	case eof:
	default:
		l.cur.col++
	}

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

func unescapeQuotes(s string, quot rune) string {
	var buf bytes.Buffer
	max := utf8.RuneCountInString(s)
	rdr := strings.NewReader(s)
	for i := 0; i < max; i++ {
		r, _, _ := rdr.ReadRune()

		// assume first rune and last rune are quot
		if i == 0 || i == max-1 {
			continue
		}

		switch r {
		case '\\', quot: // possible escape sequence
			if r2, _, _ := rdr.ReadRune(); r2 == quot {
				i++
				r = quot
			} else {
				rdr.UnreadRune()
			}
		}
		buf.WriteRune(r)
	}
	return buf.String()
}

func (l *lexer) runQuote(pair rune) error {
	for {
		r := l.next()
		if r == eof {
			return errors.New(`unexpected eof`)
		} else if r == '\\' {
			if l.peek() == pair {
				r = l.next()
			}
		} else if r == pair {
			if l.peek() == pair {
				// it is escape
				r = l.next()
			} else {
				return nil
			}
		}
	}
}

// https://dev.mysql.com/doc/refman/5.6/en/comments.html
func (l *lexer) runCComment() {
	for {
		r := l.next()
		switch r {
		case eof:
			return
		case '*':
			if l.peek() == '/' {
				l.advance()
				return
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

func (l *lexer) runNumber() {
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
}

func isSpace(r rune) bool {
	return r == ' ' || r == '\n' || r == '\t' || r == '\r'
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
