package schemalex

import (
	"bytes"
	"fmt"
	"strconv"
)

// ParseError is returned from the various `Parse` methods when an
// invalid or unsupported SQL is found. When stringified, the result
// will look something like this:
//
//    parse error: expected RPAREN at line 3 column 14
//	  "CREATE TABLE foo " <---- AROUND HERE
type ParseError struct {
	context string
	line    int
	col     int
	message string
	eof     bool
}

// Line returns the line number where the error was encountered
func (e ParseError) Line() int { return e.line }

// Col returns the column number where the error was encountered
func (e ParseError) Col() int { return e.col }

// EOF returns true if the error was encountered at EOF
func (e ParseError) EOF() bool { return e.eof }

// Message returns the actual error message
func (e ParseError) Message() string { return e.message }

// Error returns the formatted string representation of this parse error.
func (e ParseError) Error() string {
	var buf bytes.Buffer
	buf.WriteString("parse error: ")
	buf.WriteString(e.message)
	buf.WriteString(" at line ")
	buf.WriteString(strconv.Itoa(e.line))
	buf.WriteString(" column ")
	buf.WriteString(strconv.Itoa(e.col))
	if e.eof {
		buf.WriteString(" (at EOF)")
	}
	buf.WriteByte('\n')
	buf.WriteString(e.context)
	return buf.String()
}

func newParseError(ctx *parseCtx, t *Token, msg string, args ...interface{}) error {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}

	// find the closest newline before t.Pos
	var ctxbegin int
	if i := bytes.LastIndexByte(ctx.input[:t.Pos], '\n'); i > 0 {
		if len(ctx.input)-1 > i {
			ctxbegin = i + 1
		}
	}

	// if this is more than 40 chars from t.Pos, truncate it
	if t.Pos-ctxbegin > 40 {
		ctxbegin = t.Pos - 40
	}

	// We're going to append a marker here

	return &ParseError{
		context: fmt.Sprintf(`"%s" <---- AROUND HERE`, ctx.input[ctxbegin:t.Pos]),
		line:    t.Line,
		col:     t.Col,
		eof:     t.EOF,
		message: msg,
	}
}
