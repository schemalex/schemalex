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

func (l *lexer) read() (token, string) {
	l.start = l.pos

	r := l.next()

	if isSpace(r) {
		// read still space end
		return l.runSpace(), ""
	} else if isLetter(r) {
		l.backup()
		t, s := l.runIdent(), l.str()
		if _t, ok := keywordIdentMap[strings.ToUpper(s)]; ok {
			t = _t
		}
		return t, s
	} else if isDigit(r) {
		l.backup()
		return l.runNumber()
	}

	switch r {
	case eof:
		return EOF, ""
	case '`':
		return l.runQuote('`', BACKTICK_IDENT)
	case '"':
		// TODO: should smart
		t, s := l.runQuote('"', DOUBLE_QUOTE_IDENT)
		if t != DOUBLE_QUOTE_IDENT {
			return t, s
		}
		return t, `"` + s + `"`
	case '\'':
		// TODO: should smart
		t, s := l.runQuote('\'', SINGLE_QUOTE_IDENT)
		if t != SINGLE_QUOTE_IDENT {
			return t, s
		}
		return t, `'` + s + `'`
	case '/':
		if l.peek() == '*' {
			return l.runCComment(), ""
		}
		return SLASH, ""
	case '-':
		r1 := l.peek()
		if r1 == '-' {
			l.next()
			// TODO: https://dev.mysql.com/doc/refman/5.6/en/comments.html
			// TODO: not only space. control character
			if !isSpace(l.peek()) {
				l.backup()
				return DASH, ""
			}
			l.runToEOL()
			return COMMENT_IDENT, ""
		} else if isDigit(r1) {
			return l.runNumber()
		} else {
			return DASH, ""
		}
	case '#':
		// https://dev.mysql.com/doc/refman/5.6/en/comments.html
		l.runToEOL()
		return COMMENT_IDENT, ""
	case '(':
		return LPAREN, ""
	case ')':
		return RPAREN, ""
	case ';':
		return SEMICOLON, ""
	case ',':
		return COMMA, ""
	case '.':
		if isDigit(l.peek()) {
			return l.runNumber()
		}
		return DOT, ""
	case '+':
		if isDigit(l.peek()) {
			return l.runNumber()
		}
		return PLUS, ""
	case '=':
		return EQUAL, ""
	default:
		return ILLEGAL, ""
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

func (l *lexer) runSpace() token {
	for isSpace(l.peek()) {
		l.next()
	}
	return SPACE
}

func (l *lexer) runIdent() token {
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

func (l *lexer) runQuote(pair rune, t token) (token, string) {
	var b bytes.Buffer
	for {
		r := l.next()
		if r == eof {
			return ILLEGAL, ""
		} else if r == '\\' {
			if l.peek() == pair {
				r = l.next()
			}
		} else if r == pair {
			if l.peek() == pair {
				// it is escape
				r = l.next()
			} else {
				return t, b.String()
			}
		}
		b.WriteRune(r)
	}

	return ILLEGAL, ""
}

// https://dev.mysql.com/doc/refman/5.6/en/comments.html
func (l *lexer) runCComment() token {
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

func (l *lexer) runToEOL() token {
	for {
		r := l.next()
		switch r {
		case eof, '\n':
			return COMMENT_IDENT
		}
	}
}

// https://dev.mysql.com/doc/refman/5.6/en/number-literals.html
func (l *lexer) runNumber() (token, string) {

	runDigit := func() {
		for {
			r := l.next()
			if !isDigit(r) {
				l.backup()
				break
			}
		}
	}

	runDigit()

	if l.peek() == '.' {
		l.next()
		runDigit()
	}

	switch l.peek() {
	case 'E', 'e':
		l.next()
		if l.peek() == '-' {
			l.next()
		}
		runDigit()
		return NUMBER, l.str()
	default:
		return NUMBER, l.str()
	}
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
