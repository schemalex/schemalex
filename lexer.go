package schemalex

import (
	"bytes"
	"strings"
	"unicode/utf8"
)

const eof = rune(0)

type lexer struct {
	input string

	pos   int
	start int
	width int
}

func (l *lexer) str() string {
	return l.input[l.start:l.pos]
}

func (l *lexer) read() *Token {
	l.start = l.pos

	r := l.next()

	switch {
	case isSpace(r):
		// read until space end
		l.runSpace()
		return &Token{Type: SPACE}
	case isLetter(r):
		l.backup()
		t, s := l.runIdent(), l.str()
		if _t, ok := keywordIdentMap[strings.ToUpper(s)]; ok {
			t = _t
		}
		return &Token{Type: t, Value: s}
	case isDigit(r):
		l.backup()
		return l.runNumber()
	}

	switch r {
	case eof:
		return &Token{Type: EOF}
	case '`':
		return l.runQuote('`', BACKTICK_IDENT)
	case '"':
		// TODO: should smart
		t := l.runQuote('"', DOUBLE_QUOTE_IDENT)
		if t.Type != DOUBLE_QUOTE_IDENT {
			return t
		}
		t.Value = `"` + t.Value + `"`
		return t
	case '\'':
		// TODO: should smart
		t := l.runQuote('\'', SINGLE_QUOTE_IDENT)
		if t.Type != SINGLE_QUOTE_IDENT {
			return t
		}
		t.Value = `'` + t.Value + `'`
		return t
	case '/':
		if l.peek() == '*' {
			return &Token{Type: l.runCComment()}
		}
		return &Token{Type: SLASH}
	case '-':
		r1 := l.peek()
		if r1 == '-' {
			l.next()
			// TODO: https://dev.mysql.com/doc/refman/5.6/en/comments.html
			// TODO: not only space. control character
			if !isSpace(l.peek()) {
				l.backup()
				return &Token{Type: DASH}
			}
			l.runToEOL()
			return &Token{Type: COMMENT_IDENT}
		} else if isDigit(r1) {
			return l.runNumber()
		} else {
			return &Token{Type: DASH}
		}
	case '#':
		// https://dev.mysql.com/doc/refman/5.6/en/comments.html
		l.runToEOL()
		return &Token{Type: COMMENT_IDENT}
	case '(':
		return &Token{Type: LPAREN}
	case ')':
		return &Token{Type: RPAREN}
	case ';':
		return &Token{Type: SEMICOLON}
	case ',':
		return &Token{Type: COMMA}
	case '.':
		if isDigit(l.peek()) {
			return l.runNumber()
		}
		return &Token{Type: DOT}
	case '+':
		if isDigit(l.peek()) {
			return l.runNumber()
		}
		return &Token{Type: PLUS}
	case '=':
		return &Token{Type: EQUAL}
	default:
		return &Token{Type: ILLEGAL}
	}
}

func (l *lexer) next() rune {
	if l.pos >= len(l.input) {
		l.width = 0
		return eof
	}
	r, w := utf8.DecodeRuneInString(l.input[l.pos:])
	l.width = w
	l.pos += l.width

	return r
}

func (l *lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

func (l *lexer) backup() {
	l.pos -= l.width
}

func (l *lexer) runSpace() {
	for isSpace(l.peek()) {
		l.next()
	}
}

func (l *lexer) runIdent() TokenType {
	for {
		r := l.next()
		if r == eof {
			break
		} else if isCharacter(r) {
		} else {
			l.backup()
			break
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
			if l.next() == '/' {
				return COMMENT_IDENT
			}
			l.backup()
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
		r := l.next()
		if !isDigit(r) {
			l.backup()
			break
		}
	}
}

func (l *lexer) runNumber() *Token {
	l.runDigit()

	if l.peek() == '.' {
		l.next()
		l.runDigit()
	}

	switch l.peek() {
	case 'E', 'e':
		l.next()
		if l.peek() == '-' {
			l.next()
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
