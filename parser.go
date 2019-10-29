package schemalex

import (
	"context"
	"io/ioutil"
	"strings"

	"github.com/schemalex/schemalex/internal/errors"
	"github.com/schemalex/schemalex/model"
)

const (
	coloptSize = 1 << iota
	coloptDecimalSize
	coloptDecimalOptionalSize
	coloptUnsigned
	coloptZerofill
	coloptBinary
	coloptCharacterSet
	coloptCollate
	coloptEnumValues
	coloptSetValues

	// Everything else, meaning after this position, you can put anything
	// you want. e.g. these are allowed
	// * INT(11) COMMENT 'foo' NOT NULL PRIMARY KEY AUTO_INCREMENT
	// * INT(11) AUTO_INCREMENT NOT NULL DEFAULT 1
	// But this needs to be an error
	// * COMMENT 'foo' INT(11) NOT NULL

	coloptEverythingElse

	coloptNull          = coloptEverythingElse
	coloptDefault       = coloptEverythingElse
	coloptAutoIncrement = coloptEverythingElse
	coloptKey           = coloptEverythingElse
	coloptComment       = coloptEverythingElse
)

const (
	coloptFlagNone            = 0
	coloptFlagDigit           = coloptSize | coloptUnsigned | coloptZerofill
	coloptFlagDecimal         = coloptDecimalSize | coloptUnsigned | coloptZerofill
	coloptFlagDecimalOptional = coloptDecimalOptionalSize | coloptUnsigned | coloptZerofill
	coloptFlagTime            = coloptSize
	coloptFlagChar            = coloptSize | coloptBinary | coloptCharacterSet | coloptCollate
	coloptFlagBinary          = coloptSize
	coloptFlagEnum            = coloptEnumValues
	coloptFlagSet             = coloptSetValues
)

// Parser is responsible to parse a set of SQL statements
type Parser struct{}

// New creates a new Parser
func New() *Parser {
	return &Parser{}
}

type parseCtx struct {
	context.Context
	input      []byte
	lexsrc     chan *Token
	peekCount  int
	peekTokens [3]*Token
}

func newParseCtx(ctx context.Context) *parseCtx {
	return &parseCtx{
		Context:   ctx,
		peekCount: -1,
	}
}

var eofToken = Token{Type: EOF}

// peek the next token. this operation fills the peekTokens
// buffer. `next()` is a combination of peek+advance.
//
// note: we do NOT check for peekCout > 2 for efficiency.
// if you do that, you're f*cked.
func (pctx *parseCtx) peek() *Token {
	if pctx.peekCount < 0 {
		select {
		case <-pctx.Context.Done():
			return &eofToken
		case t, ok := <-pctx.lexsrc:
			if !ok {
				return &eofToken
			}
			pctx.peekCount++
			pctx.peekTokens[pctx.peekCount] = t
		}
	}
	return pctx.peekTokens[pctx.peekCount]
}

func (pctx *parseCtx) advance() {
	if pctx.peekCount >= 0 {
		pctx.peekCount--
	}
}

func (pctx *parseCtx) rewind() {
	if pctx.peekCount < 2 {
		pctx.peekCount++
	}
}

func (pctx *parseCtx) next() *Token {
	t := pctx.peek()
	pctx.advance()
	return t
}

// ParseFile parses a file containing SQL statements and creates
// a mode.Stmts structure.
// See Parse for details.
func (p *Parser) ParseFile(fn string) (model.Stmts, error) {
	src, err := ioutil.ReadFile(fn)
	if err != nil {
		return nil, errors.Wrapf(err, `failed to open file %s`, fn)
	}

	stmts, err := p.Parse(src)
	if err != nil {
		if pe, ok := err.(*parseError); ok {
			pe.file = fn
		}
		return nil, err
	}
	return stmts, nil
}

// ParseString parses a string containing SQL statements and creates
// a mode.Stmts structure.
// See Parse for details.
func (p *Parser) ParseString(src string) (model.Stmts, error) {
	return p.Parse([]byte(src))
}

// Parse parses the given set of SQL statements and creates a
// model.Stmts structure.
// If it encounters errors while parsing, the returned error will be a
// ParseError type.
func (p *Parser) Parse(src []byte) (model.Stmts, error) {
	cctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	ctx := newParseCtx(cctx)
	ctx.input = src
	ctx.lexsrc = lex(cctx, src)

	var stmts model.Stmts
LOOP:
	for {
		ctx.skipWhiteSpaces()
		switch t := ctx.peek(); t.Type {
		case CREATE:
			stmt, err := p.parseCreate(ctx)
			if err != nil {
				if errors.IsIgnorable(err) {
					// this is ignorable.
					continue
				}
				if pe, ok := err.(ParseError); ok {
					return nil, pe
				}
				return nil, errors.Wrap(err, `failed to parse create`)
			}
			stmts = append(stmts, stmt)
		case COMMENT_IDENT:
			ctx.advance()
		case DROP, SET, USE:
			// We don't do anything about these
		S1:
			for {
				switch t := ctx.peek(); t.Type {
				case SEMICOLON:
					ctx.advance()
					fallthrough
				case EOF:
					break S1
				default:
					ctx.advance()
				}
			}
		case SEMICOLON:
			// you could have statements where it's just empty, followed by a
			// semicolon. These are just empty lines, so we just skip and go
			// process the next statement
			ctx.advance()
			continue
		case EOF:
			ctx.advance()
			break LOOP
		default:
			return nil, newParseError(ctx, t, "expected CREATE, COMMENT_IDENT, SEMICOLON or EOF")
		}
	}

	return stmts, nil
}

func (p *Parser) parseCreate(ctx *parseCtx) (model.Stmt, error) {
	if t := ctx.next(); t.Type != CREATE {
		return nil, errors.New(`expected CREATE`)
	}
	ctx.skipWhiteSpaces()
	switch t := ctx.peek(); t.Type {
	case DATABASE:
		if _, err := p.parseCreateDatabase(ctx); err != nil {
			return nil, err
		}
		return nil, errors.Ignorable(nil)
	case TABLE:
		return p.parseCreateTable(ctx)
	default:
		return nil, newParseError(ctx, t, "expected DATABASE or TABLE")
	}
}

// https://dev.mysql.com/doc/refman/5.5/en/create-database.html
// TODO: charset, collation
func (p *Parser) parseCreateDatabase(ctx *parseCtx) (model.Database, error) {
	if t := ctx.next(); t.Type != DATABASE {
		return nil, errors.New(`expected DATABASE`)
	}

	ctx.skipWhiteSpaces()

	var notexists bool
	if ctx.peek().Type == IF {
		ctx.advance()
		if _, err := p.parseIdents(ctx, NOT, EXISTS); err != nil {
			return nil, err
		}
		notexists = true
	}

	ctx.skipWhiteSpaces()

	var database model.Database
	switch t := ctx.next(); t.Type {
	case IDENT, BACKTICK_IDENT:
		database = model.NewDatabase(t.Value)
	default:
		return nil, newParseError(ctx, t, "expected IDENT, BACKTICK_IDENT")
	}

	database.SetIfNotExists(notexists)
	p.eol(ctx)
	return database, nil
}

// http://dev.mysql.com/doc/refman/5.6/en/create-table.html
func (p *Parser) parseCreateTable(ctx *parseCtx) (model.Table, error) {
	if t := ctx.next(); t.Type != TABLE {
		return nil, errors.New(`expected TABLE`)
	}

	var table model.Table

	ctx.skipWhiteSpaces()
	var temporary bool
	if t := ctx.peek(); t.Type == TEMPORARY {
		ctx.advance()
		ctx.skipWhiteSpaces()
		temporary = true
	}

	// IF NOT EXISTS
	var notexists bool
	if ctx.peek().Type == IF {
		ctx.advance()
		if _, err := p.parseIdents(ctx, NOT, EXISTS); err != nil {
			return nil, err
		}
		ctx.skipWhiteSpaces()
		notexists = true
	}

	switch t := ctx.next(); t.Type {
	case IDENT, BACKTICK_IDENT:
		table = model.NewTable(t.Value)
	default:
		return nil, newParseError(ctx, t, "expected IDENT or BACKTICK_IDENT")
	}
	table.SetTemporary(temporary)
	table.SetIfNotExists(notexists)

	ctx.skipWhiteSpaces()
	switch t := ctx.peek(); t.Type {
	case LIKE:
		// CREATE TABLE foo LIKE bar
		ctx.advance()
		ctx.skipWhiteSpaces()
		switch t := ctx.next(); t.Type {
		case IDENT, BACKTICK_IDENT:
			table.SetLikeTable(t.Value)
		default:
			return nil, newParseError(ctx, t, "expected table name after LIKE")
		}

		ctx.skipWhiteSpaces()
		switch t := ctx.peek(); t.Type {
		case EOF, SEMICOLON:
			ctx.advance()
		}
		return table, nil
	case IF:
		ctx.advance()
		if _, err := p.parseIdents(ctx, NOT, EXISTS); err != nil {
			return nil, newParseError(ctx, t, "should NOT EXISTS")
		}
		ctx.skipWhiteSpaces()
		table.SetIfNotExists(true)
	}

	if t := ctx.next(); t.Type != LPAREN {
		return nil, newParseError(ctx, t, "expected LPAREN")
	}

	if err := p.parseCreateTableFields(ctx, table); err != nil {
		return nil, err
	}

	table, _ = table.Normalize()
	return table, nil
}

// Start parsing after `CREATE TABLE *** (`
func (p *Parser) parseCreateTableFields(ctx *parseCtx, stmt model.Table) error {
	for {
		ctx.skipWhiteSpaces()
		switch t := ctx.peek(); t.Type {
		case CONSTRAINT:
			if err := p.parseTableConstraint(ctx, stmt); err != nil {
				return err
			}
		case PRIMARY:
			if err := p.parseTablePrimaryKey(ctx, stmt); err != nil {
				return err
			}
		case UNIQUE:
			if err := p.parseTableUniqueKey(ctx, stmt); err != nil {
				return err
			}
		case INDEX, KEY:
			// TODO. separate to KEY and INDEX
			if err := p.parseTableIndex(ctx, stmt); err != nil {
				return err
			}
		case FULLTEXT:
			if err := p.parseTableFulltextIndex(ctx, stmt); err != nil {
				return err
			}
		case SPATIAL:
			if err := p.parseTableSpatialIndex(ctx, stmt); err != nil {
				return err
			}
		case FOREIGN:
			if err := p.parseTableForeignKey(ctx, stmt); err != nil {
				return err
			}
		case CHECK: // TODO
			return newParseError(ctx, t, "unsupported field: CHECK")
		case IDENT, BACKTICK_IDENT:
			if err := p.parseTableColumn(ctx, stmt); err != nil {
				return err
			}
		default:
			return newParseError(ctx, t, "unexpected create table field token: %s", t.Type)
		}

		ctx.skipWhiteSpaces()
		switch t := ctx.peek(); t.Type {
		case RPAREN:
			ctx.advance()
			if err := p.parseCreateTableOptions(ctx, stmt); err != nil {
				return err
			}
			// partition option
			if !p.eol(ctx) {
				return newParseError(ctx, t, "expected EOL")
			}
			return nil
		case COMMA:
			ctx.advance()
			// Expecting another table field, keep looping
		default:
			return newParseError(ctx, t, "expected RPAREN or COMMA")
		}
	}
}

func (p *Parser) parseTableConstraint(ctx *parseCtx, table model.Table) error {
	if t := ctx.next(); t.Type != CONSTRAINT {
		return newParseError(ctx, t, "expected CONSTRAINT")
	}
	ctx.skipWhiteSpaces()

	var sym string
	switch t := ctx.peek(); t.Type {
	case IDENT, BACKTICK_IDENT:
		// TODO: should be smarter
		// (lestrrat): I don't understand. How?
		sym = t.Value
		ctx.advance()
		ctx.skipWhiteSpaces()
	}

	var index model.Index
	switch t := ctx.peek(); t.Type {
	case PRIMARY:
		index = model.NewIndex(model.IndexKindPrimaryKey, table.ID())
		if err := p.parseColumnIndexPrimaryKey(ctx, index); err != nil {
			return err
		}
	case UNIQUE:
		index = model.NewIndex(model.IndexKindUnique, table.ID())
		if err := p.parseColumnIndexUniqueKey(ctx, index); err != nil {
			return err
		}
	case FOREIGN:
		index = model.NewIndex(model.IndexKindForeignKey, table.ID())
		if err := p.parseColumnIndexForeignKey(ctx, index); err != nil {
			return err
		}
	default:
		return newParseError(ctx, t, "not supported")
	}

	if len(sym) > 0 {
		index.SetSymbol(sym)
	}

	table.AddIndex(index)
	return nil
}

func (p *Parser) parseTablePrimaryKey(ctx *parseCtx, table model.Table) error {
	index := model.NewIndex(model.IndexKindPrimaryKey, table.ID())
	if err := p.parseColumnIndexPrimaryKey(ctx, index); err != nil {
		return err
	}
	table.AddIndex(index)
	return nil
}

func (p *Parser) parseTableUniqueKey(ctx *parseCtx, table model.Table) error {
	index := model.NewIndex(model.IndexKindUnique, table.ID())
	if err := p.parseColumnIndexUniqueKey(ctx, index); err != nil {
		return err
	}
	table.AddIndex(index)
	return nil
}

func (p *Parser) parseTableIndex(ctx *parseCtx, table model.Table) error {
	index := model.NewIndex(model.IndexKindNormal, table.ID())
	if err := p.parseColumnIndexKey(ctx, index); err != nil {
		return err
	}
	table.AddIndex(index)
	return nil
}

func (p *Parser) parseTableFulltextIndex(ctx *parseCtx, table model.Table) error {
	index := model.NewIndex(model.IndexKindFullText, table.ID())
	if err := p.parseColumnIndexFullTextKey(ctx, index); err != nil {
		return err
	}
	table.AddIndex(index)
	return nil
}

func (p *Parser) parseTableSpatialIndex(ctx *parseCtx, table model.Table) error {
	index := model.NewIndex(model.IndexKindSpatial, table.ID())
	if err := p.parseColumnIndexSpatialKey(ctx, index); err != nil {
		return err
	}
	table.AddIndex(index)
	return nil
}

func (p *Parser) parseTableForeignKey(ctx *parseCtx, table model.Table) error {
	index := model.NewIndex(model.IndexKindForeignKey, table.ID())
	if err := p.parseColumnIndexForeignKey(ctx, index); err != nil {
		return err
	}
	table.AddIndex(index)
	return nil
}

func (p *Parser) parseTableColumn(ctx *parseCtx, table model.Table) error {
	t := ctx.next()
	switch t.Type {
	case IDENT, BACKTICK_IDENT:
	default:
		return newParseError(ctx, t, "expcted IDENT or BACKTICK_IDENT")
	}

	col := model.NewTableColumn(t.Value)
	if err := p.parseTableColumnSpec(ctx, col); err != nil {
		return err
	}
	table.AddColumn(col)
	return nil
}

func (p *Parser) parseTableColumnSpec(ctx *parseCtx, col model.TableColumn) error {
	var coltyp model.ColumnType
	var colopt int

	ctx.skipWhiteSpaces()
	switch t := ctx.next(); t.Type {
	case BIT:
		coltyp = model.ColumnTypeBit
		colopt = coloptSize
	case TINYINT:
		coltyp = model.ColumnTypeTinyInt
		colopt = coloptFlagDigit
	case SMALLINT:
		coltyp = model.ColumnTypeSmallInt
		colopt = coloptFlagDigit
	case MEDIUMINT:
		coltyp = model.ColumnTypeMediumInt
		colopt = coloptFlagDigit
	case INT:
		coltyp = model.ColumnTypeInt
		colopt = coloptFlagDigit
	case INTEGER:
		coltyp = model.ColumnTypeInteger
		colopt = coloptFlagDigit
	case BIGINT:
		coltyp = model.ColumnTypeBigInt
		colopt = coloptFlagDigit
	case REAL:
		coltyp = model.ColumnTypeReal
		colopt = coloptFlagDecimal
	case DOUBLE:
		coltyp = model.ColumnTypeDouble
		colopt = coloptFlagDecimal
	case FLOAT:
		coltyp = model.ColumnTypeFloat
		colopt = coloptFlagDecimal
	case DECIMAL:
		coltyp = model.ColumnTypeDecimal
		colopt = coloptFlagDecimalOptional
	case NUMERIC:
		coltyp = model.ColumnTypeNumeric
		colopt = coloptFlagDecimalOptional
	case DATE:
		coltyp = model.ColumnTypeDate
		colopt = coloptFlagNone
	case TIME:
		coltyp = model.ColumnTypeTime
		colopt = coloptFlagTime
	case TIMESTAMP:
		coltyp = model.ColumnTypeTimestamp
		colopt = coloptFlagTime
	case DATETIME:
		coltyp = model.ColumnTypeDateTime
		colopt = coloptFlagTime
	case YEAR:
		coltyp = model.ColumnTypeYear
		colopt = coloptFlagNone
	case CHAR:
		coltyp = model.ColumnTypeChar
		colopt = coloptFlagChar
	case VARCHAR:
		coltyp = model.ColumnTypeVarChar
		colopt = coloptFlagChar
	case BINARY:
		coltyp = model.ColumnTypeBinary
		colopt = coloptFlagBinary
	case VARBINARY:
		coltyp = model.ColumnTypeVarBinary
		colopt = coloptFlagBinary
	case TINYBLOB:
		coltyp = model.ColumnTypeTinyBlob
		colopt = coloptFlagNone
	case BLOB:
		coltyp = model.ColumnTypeBlob
		colopt = coloptFlagNone
	case MEDIUMBLOB:
		coltyp = model.ColumnTypeMediumBlob
		colopt = coloptFlagNone
	case LONGBLOB:
		coltyp = model.ColumnTypeLongBlob
		colopt = coloptFlagNone
	case TINYTEXT:
		coltyp = model.ColumnTypeTinyText
		colopt = coloptFlagChar
	case TEXT:
		coltyp = model.ColumnTypeText
		colopt = coloptFlagChar
	case MEDIUMTEXT:
		coltyp = model.ColumnTypeMediumText
		colopt = coloptFlagChar
	case LONGTEXT:
		coltyp = model.ColumnTypeLongText
		colopt = coloptFlagChar
	case ENUM:
		coltyp = model.ColumnTypeEnum
		colopt = coloptFlagEnum
	case SET:
		coltyp = model.ColumnTypeSet
		colopt = coloptFlagSet
	case BOOLEAN:
		coltyp = model.ColumnTypeBoolean
		colopt = coloptFlagNone
	case BOOL:
		coltyp = model.ColumnTypeBool
		colopt = coloptFlagNone
	case JSON:
		coltyp = model.ColumnTypeJSON
		colopt = coloptFlagNone
	default:
		return newParseError(ctx, t, "unsupported type in column specification")
	}

	col.SetType(coltyp)
	return p.parseColumnOption(ctx, col, colopt)
}

func (p *Parser) parseCreateTableOptionValue(ctx *parseCtx, table model.Table, name string, follow ...TokenType) error {
	ctx.skipWhiteSpaces()
	if t := ctx.peek(); t.Type == EQUAL {
		ctx.advance()
		ctx.skipWhiteSpaces()
	}

	t := ctx.next()
	for _, typ := range follow {
		if typ != t.Type {
			continue
		}
		var quotes bool
		switch t.Type {
		case SINGLE_QUOTE_IDENT, DOUBLE_QUOTE_IDENT:
			quotes = true
		}
		table.AddOption(model.NewTableOption(name, t.Value, quotes))
		return nil
	}
	return newParseError(ctx, t, "expected %v", follow)
}

func (p *Parser) parseCreateTableOptions(ctx *parseCtx, table model.Table) error {
	ctx.skipWhiteSpaces()
	switch t := ctx.peek(); t.Type {
	case EOF:
		// no table options, end of input
		ctx.advance()
		return nil
	case SEMICOLON:
		// no table options, end of statement
		return nil
	}

	for {
		ctx.skipWhiteSpaces()
		switch t := ctx.next(); t.Type {
		case ENGINE:
			if err := p.parseCreateTableOptionValue(ctx, table, "ENGINE", IDENT, BACKTICK_IDENT); err != nil {
				return err
			}
		case AUTO_INCREMENT:
			if err := p.parseCreateTableOptionValue(ctx, table, "AUTO_INCREMENT", NUMBER); err != nil {
				return err
			}
		case AVG_ROW_LENGTH:
			if err := p.parseCreateTableOptionValue(ctx, table, "AVG_ROW_LENGTH", NUMBER); err != nil {
				return err
			}
		case DEFAULT:
			var name string
			ctx.skipWhiteSpaces()
			switch t := ctx.next(); t.Type {
			case CHARSET:
				name = "DEFAULT CHARACTER SET"
			case CHARACTER:
				ctx.skipWhiteSpaces()
				if t := ctx.next(); t.Type != SET {
					return newParseError(ctx, t, "expected SET")
				}
				name = "DEFAULT CHARACTER SET"
			case COLLATE:
				name = "DEFAULT COLLATE"
			default:
				return newParseError(ctx, t, "expected CHARACTER or COLLATE")
			}
			if err := p.parseCreateTableOptionValue(ctx, table, name, IDENT, BACKTICK_IDENT); err != nil {
				return err
			}
		case CHARACTER:
			ctx.skipWhiteSpaces()
			if t := ctx.next(); t.Type != SET {
				return newParseError(ctx, t, "expected SET")
			}
			if err := p.parseCreateTableOptionValue(ctx, table, "DEFAULT CHARACTER SET", IDENT, BACKTICK_IDENT); err != nil {
				return err
			}
		case COLLATE:
			if err := p.parseCreateTableOptionValue(ctx, table, "DEFAULT COLLATE", IDENT, BACKTICK_IDENT); err != nil {
				return err
			}
		case CHECKSUM:
			if err := p.parseCreateTableOptionValue(ctx, table, "CHECKSUM", NUMBER); err != nil {
				return err
			}
		case COMMENT:
			if err := p.parseCreateTableOptionValue(ctx, table, "COMMENT", SINGLE_QUOTE_IDENT, DOUBLE_QUOTE_IDENT); err != nil {
				return err
			}
		case CONNECTION:
			if err := p.parseCreateTableOptionValue(ctx, table, "CONNECTION", SINGLE_QUOTE_IDENT, DOUBLE_QUOTE_IDENT); err != nil {
				return err
			}
		case DATA:
			ctx.skipWhiteSpaces()
			if t := ctx.next(); t.Type != DIRECTORY {
				return newParseError(ctx, t, "expected DIRECTORY")
			}
			if err := p.parseCreateTableOptionValue(ctx, table, "DATA DIRECTORY", SINGLE_QUOTE_IDENT, DOUBLE_QUOTE_IDENT); err != nil {
				return err
			}
		case DELAY_KEY_WRITE:
			if err := p.parseCreateTableOptionValue(ctx, table, "DATA_KEY_WRITE", NUMBER); err != nil {
				return err
			}
		case INDEX:
			ctx.skipWhiteSpaces()
			if t := ctx.next(); t.Type != DIRECTORY {
				return newParseError(ctx, t, "should DIRECTORY")
			}
			if err := p.parseCreateTableOptionValue(ctx, table, "INDEX DIRECTORY", SINGLE_QUOTE_IDENT, DOUBLE_QUOTE_IDENT); err != nil {
				return err
			}
		case INSERT_METHOD:
			if err := p.parseCreateTableOptionValue(ctx, table, "INSERT_METHOD", IDENT); err != nil {
				return err
			}
		case KEY_BLOCK_SIZE:
			if err := p.parseCreateTableOptionValue(ctx, table, "KEY_BLOCK_SIZE", NUMBER); err != nil {
				return err
			}
		case MAX_ROWS:
			if err := p.parseCreateTableOptionValue(ctx, table, "MAX_ROWS", NUMBER); err != nil {
				return err
			}
		case MIN_ROWS:
			if err := p.parseCreateTableOptionValue(ctx, table, "MIN_ROWS", NUMBER); err != nil {
				return err
			}
		case PACK_KEYS:
			if err := p.parseCreateTableOptionValue(ctx, table, "PACK_KEYS", NUMBER, IDENT); err != nil {
				return err
			}
		case PASSWORD:
			if err := p.parseCreateTableOptionValue(ctx, table, "PASSWORD", SINGLE_QUOTE_IDENT, DOUBLE_QUOTE_IDENT); err != nil {
				return err
			}
		case ROW_FORMAT:
			if err := p.parseCreateTableOptionValue(ctx, table, "ROW_FORMAT", DEFAULT, DYNAMIC, FIXED, COMPRESSED, REDUNDANT, COMPACT); err != nil {
				return err
			}
		case STATS_AUTO_RECALC:
			if err := p.parseCreateTableOptionValue(ctx, table, "STATS_AUTO_RECALC", NUMBER, DEFAULT); err != nil {
				return err
			}
		case STATS_PERSISTENT:
			if err := p.parseCreateTableOptionValue(ctx, table, "STATS_PERSISTENT", NUMBER, DEFAULT); err != nil {
				return err
			}
		case STATS_SAMPLE_PAGES:
			if err := p.parseCreateTableOptionValue(ctx, table, "STATS_SAMPLE_PAGES", NUMBER); err != nil {
				return err
			}
		case TABLESPACE:
			return newParseError(ctx, t, "unsupported option TABLESPACE")
		case UNION:
			return newParseError(ctx, t, "unsupported option UNION")
		case COMMA:
			// no op, continue to next option
			continue
		default:
			return newParseError(ctx, t, "unexpected token in table options: "+t.Type.String())
		}

		ctx.skipWhiteSpaces()
		// except for the case where we continue to the next option (COMMA)
		// we should expect the end of this statement
		switch t := ctx.peek(); t.Type {
		case EOF:
			// end of table options, end of input
			ctx.advance()
			return nil
		case SEMICOLON:
			// end of table options, end of statement
			return nil
		}
	}
}

// parse column options
//
// Also see: https://github.com/schemalex/schemalex/pull/40
// Seems like MySQL doesn't really care about the order of some elements in the
// column options, although the docs (https://dev.mysql.com/doc/refman/5.7/en/create-table.html)
// seem to state otherwise.
//
func (p *Parser) parseColumnOption(ctx *parseCtx, col model.TableColumn, f int) error {
	f = f | coloptNull | coloptDefault | coloptAutoIncrement | coloptKey | coloptComment
	pos := 0
	check := func(_f int) bool {
		if pos > _f {
			return false
		}
		if f|_f != f {
			return false
		}
		pos = _f
		return true
	}
	for {
		ctx.skipWhiteSpaces()
		switch t := ctx.next(); t.Type {
		case LPAREN:
			if check(coloptSize) {
				ctx.skipWhiteSpaces()
				t := ctx.next()
				if t.Type != NUMBER {
					return newParseError(ctx, t, "expected NUMBER (column size)")
				}
				tlen := t.Value

				ctx.skipWhiteSpaces()
				t = ctx.next()
				if t.Type != RPAREN {
					return newParseError(ctx, t, "expected RPAREN (column size)")
				}
				col.SetLength(model.NewLength(tlen))
			} else if check(coloptDecimalSize) {
				strs, err := p.parseIdents(ctx, NUMBER, COMMA, NUMBER, RPAREN)
				if err != nil {
					return err
				}
				l := model.NewLength(strs[0])
				l.SetDecimal(strs[2])
				col.SetLength(l)
			} else if check(coloptDecimalOptionalSize) {
				ctx.skipWhiteSpaces()
				t := ctx.next()
				if t.Type != NUMBER {
					return newParseError(ctx, t, "expected NUMBER (decimal size `M`)")
				}
				tlen := t.Value

				ctx.skipWhiteSpaces()
				t = ctx.next()
				if t.Type == RPAREN {
					col.SetLength(model.NewLength(tlen))
					continue
				} else if t.Type != COMMA {
					return newParseError(ctx, t, "expected COMMA (decimal size)")
				}

				ctx.skipWhiteSpaces()
				t = ctx.next()
				if t.Type != NUMBER {
					return newParseError(ctx, t, "expected NUMBER (decimal size `D`)")
				}
				tscale := t.Value

				ctx.skipWhiteSpaces()
				if t := ctx.next(); t.Type != RPAREN {
					return newParseError(ctx, t, "expected RPAREN (decimal size)")
				}
				l := model.NewLength(tlen)
				l.SetDecimal(tscale)
				col.SetLength(l)
			} else if check(coloptEnumValues) {
				ctx.parseSetOrEnum(col.SetEnumValues)
			} else if check(coloptSetValues) {
				ctx.parseSetOrEnum(col.SetSetValues)
			} else {
				return newParseError(ctx, t, "cannot apply coloptSize, coloptDecimalSize, coloptDecimalOptionalSize, coloptEnumValues, coloptSetValues")
			}
		case CHARACTER:
			ctx.skipWhiteSpaces()
			if t := ctx.next(); t.Type != SET {
				return newParseError(ctx, t, "expected SET")
			}
			ctx.skipWhiteSpaces()
			v := ctx.next()
			col.SetCharacterSet(v.Value)
		case COLLATE:
			ctx.skipWhiteSpaces()
			v := ctx.next()
			col.SetCollation(v.Value)
		case UNSIGNED:
			if !check(coloptUnsigned) {
				return newParseError(ctx, t, "cannot apply UNSIGNED")
			}
			col.SetUnsigned(true)
		case ZEROFILL:
			if !check(coloptZerofill) {
				return newParseError(ctx, t, "cannot apply ZEROFILL")
			}
			col.SetZeroFill(true)
		case BINARY:
			if !check(coloptBinary) {
				return newParseError(ctx, t, "cannot apply BINARY")
			}
			col.SetBinary(true)
		case NOT:
			if !check(coloptNull) {
				return newParseError(ctx, t, "cannot apply NOT NULL")
			}
			ctx.skipWhiteSpaces()
			switch t := ctx.next(); t.Type {
			case NULL:
				col.SetNullState(model.NullStateNotNull)
			default:
				return newParseError(ctx, t, "expected NULL")
			}
		case NULL:
			if !check(coloptNull) {
				return newParseError(ctx, t, "cannot apply NULL")
			}
			col.SetNullState(model.NullStateNull)
		case ON:
			// for now, only applicable to ON UPDATE ...
			ctx.skipWhiteSpaces()
			if t := ctx.next(); t.Type != UPDATE {
				return newParseError(ctx, t, "expected ON UPDATE")
			}
			ctx.skipWhiteSpaces()
			v := ctx.next()
			col.SetAutoUpdate(v.Value)
		case DEFAULT:
			if !check(coloptDefault) {
				return newParseError(ctx, t, "cannot apply DEFAULT")
			}
			ctx.skipWhiteSpaces()
			switch t := ctx.next(); t.Type {
			case IDENT, SINGLE_QUOTE_IDENT, DOUBLE_QUOTE_IDENT:
				col.SetDefault(t.Value, true)
			case NUMBER, CURRENT_TIMESTAMP, NULL, TRUE, FALSE:
				col.SetDefault(strings.ToUpper(t.Value), false)
			case NOW:
				now := t.Value
				if t := ctx.next(); t.Type != LPAREN {
					return newParseError(ctx, t, "expected LPAREN")
				}
				if t := ctx.next(); t.Type != RPAREN {
					return newParseError(ctx, t, "expected RPAREN")
				}
				col.SetDefault(strings.ToUpper(now)+"()", false)
			default:
				return newParseError(ctx, t, "expected IDENT, SINGLE_QUOTE_IDENT, DOUBLE_QUOTE_IDENT, NUMBER, CURRENT_TIMESTAMP, NULL")
			}
		case AUTO_INCREMENT:
			if !check(coloptAutoIncrement) {
				return newParseError(ctx, t, "cannot apply AUTO_INCREMENT")
			}
			col.SetAutoIncrement(true)
		case UNIQUE:
			if !check(coloptKey) {
				return newParseError(ctx, t, "cannot apply UNIQUE KEY")
			}
			ctx.skipWhiteSpaces()
			if t := ctx.peek(); t.Type == KEY {
				ctx.advance()
			}
			col.SetUnique(true)
		case KEY:
			if !check(coloptKey) {
				return newParseError(ctx, t, "cannot apply KEY")
			}
			col.SetKey(true)
		case PRIMARY:
			if !check(coloptKey) {
				return newParseError(ctx, t, "cannot apply PRIMARY KEY")
			}
			ctx.skipWhiteSpaces()
			if t := ctx.next(); t.Type != KEY {
				return newParseError(ctx, t, "expected PRIMARY KEY")
			}
			col.SetPrimary(true)
		case COMMENT:
			if !check(coloptComment) {
				return newParseError(ctx, t, "cannot apply COMMENT")
			}
			ctx.skipWhiteSpaces()
			switch t := ctx.next(); t.Type {
			case SINGLE_QUOTE_IDENT:
				col.SetComment(t.Value)
			default:
				return newParseError(ctx, t, "should SINGLE_QUOTE_IDENT")
			}
		case COMMA:
			ctx.rewind()
			return nil
		case RPAREN:
			ctx.rewind()
			return nil
		default:
			return newParseError(ctx, t, "unexpected column option %s", t.Type)
		}
	}
}

func (ctx *parseCtx) parseSetOrEnum(setter func([]string) model.TableColumn) error {
	var values []string
OUTER:
	for {
		ctx.skipWhiteSpaces()
		v := ctx.next()
		if v.Type == SINGLE_QUOTE_IDENT || v.Type == DOUBLE_QUOTE_IDENT {
			values = append(values, v.Value)
		} else {
			return newParseError(ctx, v, "expected RPAREN, SINGLE_QUOTE_IDENT, DOUBLE_QUOTE_IDENT(enum values): %s", v.Type)
		}
		ctx.skipWhiteSpaces()

		switch t := ctx.next(); t.Type {
		case COMMA:
		case RPAREN:
			break OUTER
		default:
			return newParseError(ctx, t, "expected COMMA")
		}
	}
	setter(values)
	return nil
}

func (p *Parser) parseColumnIndexPrimaryKey(ctx *parseCtx, index model.Index) error {
	ctx.skipWhiteSpaces()
	if t := ctx.next(); t.Type != PRIMARY {
		return newParseError(ctx, t, "expected PRIMARY")
	}
	ctx.skipWhiteSpaces()
	if t := ctx.next(); t.Type != KEY {
		return newParseError(ctx, t, "expected KEY")
	}

	return p.parseColumnIndexCommon(ctx, index)
}

func (p *Parser) parseColumnIndexUniqueKey(ctx *parseCtx, index model.Index) error {
	ctx.skipWhiteSpaces()
	switch t := ctx.next(); t.Type {
	case UNIQUE:
	default:
		return newParseError(ctx, t, "expected UNIQUE")
	}

	ctx.skipWhiteSpaces()
	switch t := ctx.peek(); t.Type {
	case KEY, INDEX:
		ctx.advance()
	}

	return p.parseColumnIndexCommon(ctx, index)
}

func (p *Parser) parseColumnIndexCommon(ctx *parseCtx, index model.Index) error {
	if err := p.parseColumnIndexName(ctx, index); err != nil {
		return err
	}

	if err := p.parseColumnIndexType(ctx, index); err != nil {
		return err
	}

	if err := p.parseColumnIndexColumns(ctx, index); err != nil {
		return err
	}

	// Doing this AGAIN, because apparently you can specify the index_type
	// before or after the column declarations
	if err := p.parseColumnIndexType(ctx, index); err != nil {
		return err
	}

	return nil
}

func (p *Parser) parseColumnIndexKey(ctx *parseCtx, index model.Index) error {
	switch t := ctx.next(); t.Type {
	case KEY, INDEX:
		ctx.advance()
	default:
		return newParseError(ctx, t, "expected KEY or INDEX")
	}

	return p.parseColumnIndexCommon(ctx, index)
}

func (p *Parser) parseColumnIndexFullTextKey(ctx *parseCtx, index model.Index) error {
	ctx.skipWhiteSpaces()
	if t := ctx.next(); t.Type != FULLTEXT {
		return newParseError(ctx, t, "expected FULLTEXT")
	}

	// optional INDEX
	ctx.skipWhiteSpaces()
	if t := ctx.peek(); t.Type == INDEX {
		ctx.advance()
	}

	if err := p.parseColumnIndexName(ctx, index); err != nil {
		return err
	}

	if err := p.parseColumnIndexColumns(ctx, index); err != nil {
		return err
	}

	return nil
}

func (p *Parser) parseColumnIndexSpatialKey(ctx *parseCtx, index model.Index) error {
	ctx.skipWhiteSpaces()
	if t := ctx.next(); t.Type != SPATIAL {
		return newParseError(ctx, t, "expected SPATIAL")
	}

	// optional INDEX
	ctx.skipWhiteSpaces()
	if t := ctx.peek(); t.Type == INDEX {
		ctx.advance()
	}

	if err := p.parseColumnIndexName(ctx, index); err != nil {
		return err
	}

	if err := p.parseColumnIndexColumns(ctx, index); err != nil {
		return err
	}

	return nil
}

func (p *Parser) parseColumnIndexForeignKey(ctx *parseCtx, index model.Index) error {
	ctx.skipWhiteSpaces()
	if t := ctx.next(); t.Type != FOREIGN {
		return newParseError(ctx, t, "expected FOREGIN")
	}

	ctx.skipWhiteSpaces()
	if t := ctx.next(); t.Type != KEY {
		return newParseError(ctx, t, "expected KEY")
	}
	if err := p.parseColumnIndexName(ctx, index); err != nil {
		return err
	}

	if err := p.parseColumnIndexColumns(ctx, index); err != nil {
		return err
	}

	ctx.skipWhiteSpaces()
	if t := ctx.peek(); t.Type == REFERENCES {
		if err := p.parseColumnReference(ctx, index); err != nil {
			return err
		}
	}

	return nil
}

func (p *Parser) parseReferenceOption(ctx *parseCtx, set func(model.ReferenceOption) model.Reference) error {
	ctx.skipWhiteSpaces()
	switch t := ctx.next(); t.Type {
	case RESTRICT:
		set(model.ReferenceOptionRestrict)
	case CASCADE:
		set(model.ReferenceOptionCascade)
	case SET:
		ctx.skipWhiteSpaces()
		if t := ctx.next(); t.Type != NULL {
			return newParseError(ctx, t, "expected NULL")
		}
		set(model.ReferenceOptionSetNull)
	case NO:
		ctx.skipWhiteSpaces()
		if t := ctx.next(); t.Type != ACTION {
			return newParseError(ctx, t, "expected ACTION")
		}
		set(model.ReferenceOptionNoAction)
	default:
		return newParseError(ctx, t, "expected RESTRICT, CASCADE, SET or NO")
	}
	return nil
}

func (p *Parser) parseColumnReference(ctx *parseCtx, index model.Index) error {
	ctx.skipWhiteSpaces()
	if t := ctx.next(); t.Type != REFERENCES {
		return newParseError(ctx, t, "expected REFERENCES")
	}

	r := model.NewReference()

	ctx.skipWhiteSpaces()
	switch t := ctx.next(); t.Type {
	case BACKTICK_IDENT, IDENT:
		r.SetTableName(t.Value)
	default:
		return newParseError(ctx, t, "expected IDENT or BACKTICK_IDENT")
	}

	if err := p.parseColumnIndexColumns(ctx, r); err != nil {
		return err
	}

	ctx.skipWhiteSpaces()
	if t := ctx.peek(); t.Type == MATCH {
		ctx.advance()
		ctx.skipWhiteSpaces()
		switch t = ctx.next(); t.Type {
		case FULL:
			r.SetMatch(model.ReferenceMatchFull)
		case PARTIAL:
			r.SetMatch(model.ReferenceMatchPartial)
		case SIMPLE:
			r.SetMatch(model.ReferenceMatchSimple)
		default:
			return newParseError(ctx, t, "should FULL, PARTIAL or SIMPLE")
		}
		ctx.skipWhiteSpaces()
	}

	// ON DELETE can be followed by ON UPDATE, but
	// ON UPDATE cannot be followed by ON DELETE
OUTER:
	for i := 0; i < 2; i++ {
		ctx.skipWhiteSpaces()
		if t := ctx.peek(); t.Type != ON {
			break OUTER
		}
		ctx.advance()
		ctx.skipWhiteSpaces()

		switch t := ctx.next(); t.Type {
		case DELETE:
			if err := p.parseReferenceOption(ctx, r.SetOnDelete); err != nil {
				return errors.Wrap(err, `failed to parse ON DELETE`)
			}
		case UPDATE:
			if err := p.parseReferenceOption(ctx, r.SetOnUpdate); err != nil {
				return errors.Wrap(err, `failed to parse ON UPDATE`)
			}
			break OUTER
		default:
			return newParseError(ctx, t, "expected DELETE or UPDATE")
		}
	}

	index.SetReference(r)
	return nil
}

func (p *Parser) parseColumnIndexName(ctx *parseCtx, index model.Index) error {
	ctx.skipWhiteSpaces()
	switch t := ctx.peek(); t.Type {
	case BACKTICK_IDENT, IDENT:
		ctx.advance()
		index.SetName(t.Value)
	}
	return nil
}

func (p *Parser) parseColumnIndexType(ctx *parseCtx, index model.Index) error {
	ctx.skipWhiteSpaces()
	if t := ctx.peek(); t.Type != USING {
		return nil
	}
	ctx.advance()

	if index.HasType() {
		var typ string
		if index.IsBtree() {
			typ = "BTREE"
		} else if index.IsHash() {
			typ = "HASH"
		} else {
			typ = "NONE"
		}
		return errors.Errorf(`statement already has index type declared (%s)`, typ)
	}

	ctx.skipWhiteSpaces()
	switch t := ctx.next(); t.Type {
	case BTREE:
		index.SetType(model.IndexTypeBtree)
	case HASH:
		index.SetType(model.IndexTypeHash)
	default:
		return newParseError(ctx, t, "expected BTREE or HASH")
	}
	return nil
}

// TODO rename method name
func (p *Parser) parseColumnIndexColumns(ctx *parseCtx, container interface {
	AddColumns(...model.IndexColumn)
}) error {

	var cols []model.IndexColumn

	ctx.skipWhiteSpaces()
	if t := ctx.next(); t.Type != LPAREN {
		return newParseError(ctx, t, "expected LPAREN while parsing index column: %s", t.Type)
	}

OUTER:
	for {
		ctx.skipWhiteSpaces()
		t := ctx.next()
		if !(t.Type == IDENT || t.Type == BACKTICK_IDENT) {
			return newParseError(ctx, t, "should IDENT or BACKTICK_IDENT")
		}

		col := model.NewIndexColumn(t.Value)
		cols = append(cols, col)

		ctx.skipWhiteSpaces()
		switch t = ctx.next(); t.Type {
		case LPAREN:
			t := ctx.next()
			if t.Type != NUMBER {
				return newParseError(ctx, t, "expected NUMBER")
			}
			tlen := t.Value
			ctx.skipWhiteSpaces()
			if t = ctx.next(); t.Type != RPAREN {
				return newParseError(ctx, t, "expected RPAREN")
			}
			col.SetLength(tlen)
		default:
			ctx.rewind()
		}

		// optional sort direction
		switch t = ctx.peek(); t.Type {
		case ASC:
			ctx.advance()
			col.SetSortDirection(model.SortDirectionAscending)
		case DESC:
			ctx.advance()
			col.SetSortDirection(model.SortDirectionDescending)
		}

		ctx.skipWhiteSpaces()
		switch t = ctx.next(); t.Type {
		case COMMA:
			// search next
			continue
		case RPAREN:
			break OUTER
		default:
			return newParseError(ctx, t, "expected COMMA or RPAREN")
		}
	}

	container.AddColumns(cols...)
	return nil
}

// Skips over whitespaces. Once this method returns, you can be
// certain that next call to ctx.next()/peek() will result in a
// non-space token
func (pctx *parseCtx) skipWhiteSpaces() {
	for {
		switch t := pctx.peek(); t.Type {
		case SPACE, COMMENT_IDENT:
			pctx.advance()
			continue
		default:
			return
		}
	}
}

func (p *Parser) parseIdents(ctx *parseCtx, idents ...TokenType) ([]string, error) {
	strs := []string{}
	for _, ident := range idents {
		ctx.skipWhiteSpaces()
		t := ctx.next()
		if t.Type != ident {
			return nil, newParseError(ctx, t, "expected %v", idents)
		}
		strs = append(strs, t.Value)
	}
	return strs, nil
}

// TODO: revisit what exactly this eol is meant to do
func (p *Parser) eol(ctx *parseCtx) bool {
	ctx.skipWhiteSpaces()
	switch t := ctx.next(); t.Type {
	case EOF, SEMICOLON:
		ctx.advance()
		return true
	default:
		return false
	}
}
