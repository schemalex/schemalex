package schemalex

import (
	"fmt"
	"io/ioutil"
	"math"

	"github.com/schemalex/schemalex/internal/errors"
	"golang.org/x/net/context"
)

type Parser struct {
	ErrorMarker  string
	ErrorContext int
}

func New() *Parser {
	return &Parser{}
}

type parseCtx struct {
	context.Context
	lexer        lexer // TODO delete
	lexsrc       chan *Token
	errorMarker  string
	errorContext int
	peekCount    int
	peekTokens   [3]*Token
}

func newParseCtx(ctx context.Context) *parseCtx {
	return &parseCtx{
		Context:   ctx,
		peekCount: -1,
	}
}

var eofToken = Token{Type: EOF}

// peek the next token. if already
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

func (p *Parser) ParseFile(fn string) (Statements, error) {
	src, err := ioutil.ReadFile(fn)
	if err != nil {
		return nil, errors.Wrapf(err, `failed to open file %s`, fn)
	}
	return p.Parse(src)
}

func (p *Parser) ParseString(src string) (Statements, error) {
	return p.Parse([]byte(src))
}

func (p *Parser) Parse(src []byte) (Statements, error) {
	cctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	ctx := newParseCtx(cctx)
	ctx.lexsrc = Lex(cctx, src)
	ctx.errorMarker = p.ErrorMarker
	ctx.errorContext = p.ErrorContext

	var stmts []Stmt
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
				return nil, errors.Wrap(err, `failed to parse create`)
			}
			stmts = append(stmts, stmt)
		case COMMENT_IDENT:
			ctx.advance()
		case DROP, SET, USE:
			// We don't do anything about these
	S1:
			for {
				if p.eol(ctx) {
					break S1
				}
			}
		case EOF:
			ctx.advance()
			break LOOP
		default:
			return nil, p.parseErrorf(ctx, "should CREATE, COMMENT_IDENT or EOF")
		}
	}

	return stmts, nil
}

func (p *Parser) parseCreate(ctx *parseCtx) (Stmt, error) {
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
		return nil, p.parseErrorf(ctx, "should DATABASE or TABLE")
	}
}

// https://dev.mysql.com/doc/refman/5.5/en/create-database.html
// TODO: charset, collation
func (p *Parser) parseCreateDatabase(ctx *parseCtx) (*CreateDatabaseStatement, error) {
	if t := ctx.next(); t.Type != DATABASE {
		return nil, errors.New(`expected DATABASE`)
	}

	var stmt CreateDatabaseStatement
	var t *Token
	setname := func() error {
		switch t.Type {
		case IDENT, BACKTICK_IDENT:
			stmt.Name = t.Value
		default:
			return p.parseErrorf(ctx, "should IDENT or BACKTICK_IDENT")
		}
		if p.eol(ctx) {
			return nil
		}
		return p.parseErrorf(ctx, "should EOL")
	}

	ctx.skipWhiteSpaces()
	switch t = ctx.next(); t.Type {
	case IDENT, BACKTICK_IDENT:
		if err := setname(); err != nil {
			return nil, err
		}
		return &stmt, nil
	case IF:
		if _, err := p.parseIdents(ctx, NOT, EXISTS); err != nil {
			return nil, err
		}
		ctx.skipWhiteSpaces()
		t = ctx.next()
		stmt.IfNotExist = true
		if err := setname(); err != nil {
			return nil, err
		}
		return &stmt, nil
	default:
		return nil, p.parseErrorf(ctx, "should IDENT, BACKTICK_IDENT or IF")
	}
}

// http://dev.mysql.com/doc/refman/5.6/en/create-table.html
func (p *Parser) parseCreateTable(ctx *parseCtx) (*CreateTableStatement, error) {
	if t := ctx.next(); t.Type != TABLE {
		return nil, errors.New(`expected TABLE`)
	}

	var stmt CreateTableStatement

	ctx.skipWhiteSpaces()
	if t := ctx.peek(); t.Type == TEMPORARY {
		ctx.advance()
		ctx.skipWhiteSpaces()
		stmt.Temporary = true
	}

	switch t := ctx.next(); t.Type {
	case IDENT, BACKTICK_IDENT:
		stmt.Name = t.Value
	default:
		return nil, p.parseErrorf(ctx, "expected IDENT or BACKTICK_IDENT")
	}

	ctx.skipWhiteSpaces()
	if t := ctx.peek(); t.Type == IF {
		ctx.advance()
		if _, err := p.parseIdents(ctx, NOT, EXISTS); err != nil {
			return nil, p.parseErrorf(ctx, "should NOT EXISTS")
		}
		ctx.skipWhiteSpaces()
		stmt.IfNotExist = true
	}

	if t := ctx.next(); t.Type != LPAREN {
		return nil, p.parseErrorf(ctx, "should (")
	}

	if err := p.parseCreateTableFields(ctx, &stmt); err != nil {
		return nil, err
	}

	return &stmt, nil
}

// Start parsing after `CREATE TABLE *** (`
func (p *Parser) parseCreateTableFields(ctx *parseCtx, stmt *CreateTableStatement) error {
	var targetStmt interface{}

	appendStmt := func() {
		switch t := targetStmt.(type) {
		case *CreateTableIndexStatement:
			stmt.Indexes = append(stmt.Indexes, t)
		case *CreateTableColumnStatement:
			stmt.Columns = append(stmt.Columns, t)
		default:
			panic(fmt.Sprintf("unexpected targetStmt: %#v", t))
		}
		targetStmt = nil
	}

	setStmt := func(f func() (interface{}, error)) error {
		if targetStmt != nil {
			return p.parseErrorf(ctx, "seems not to be end previous column or index definition")
		}
		stmt, err := f()
		if err != nil {
			return err
		}
		targetStmt = stmt
		return nil
	}

	for {
		ctx.skipWhiteSpaces()
		switch t:= ctx.next(); t.Type {
		case RPAREN:
			appendStmt()
			if err := p.parseCreateTableOptions(ctx, stmt); err != nil {
				return err
			}
			// partition option
			if !p.eol(ctx) {
				return p.parseErrorf(ctx, "should EOL")
			}
			return nil
		case COMMA:
			if targetStmt == nil {
				return p.parseErrorf(ctx, "unexpected COMMA")
			}
			appendStmt()
		case CONSTRAINT:
			err := setStmt(func() (interface{}, error) {
				var indexStmt CreateTableIndexStatement
				ctx.skipWhiteSpaces()
				switch t := ctx.peek(); t.Type {
				case IDENT, BACKTICK_IDENT:
					// TODO: should smart
					copyStr := t.Value
					indexStmt.Symbol.Valid = true
					indexStmt.Symbol.Value = copyStr
					ctx.skipWhiteSpaces()
				}

				switch t := ctx.next(); t.Type {
				case PRIMARY:
					indexStmt.Kind = IndexKindPrimaryKey
					if err := p.parseColumnIndexPrimaryKey(ctx, &indexStmt); err != nil {
						return nil, err
					}
				case UNIQUE:
					indexStmt.Kind = IndexKindUnique
					if err := p.parseColumnIndexUniqueKey(ctx, &indexStmt); err != nil {
						return nil, err
					}
				case FOREIGN:
					indexStmt.Kind = IndexKindForeignKey
					if err := p.parseColumnIndexForeignKey(ctx, &indexStmt); err != nil {
						return nil, err
					}
				default:
					return nil, p.parseErrorf(ctx, "not supported")
				}
				return &indexStmt, nil
			})
			if err != nil {
				return err
			}
		case PRIMARY:
			err := setStmt(func() (interface{}, error) {
				var indexStmt CreateTableIndexStatement
				indexStmt.Kind = IndexKindPrimaryKey
				if err := p.parseColumnIndexPrimaryKey(ctx, &indexStmt); err != nil {
					return nil, err
				}
				return &indexStmt, nil
			})
			if err != nil {
				return err
			}
		case UNIQUE:
			err := setStmt(func() (interface{}, error) {
				indexStmt := CreateTableIndexStatement{}
				indexStmt.Kind = IndexKindUnique
				if err := p.parseColumnIndexUniqueKey(ctx, &indexStmt); err != nil {
					return nil, err
				}
				return &indexStmt, nil
			})
			if err != nil {
				return err
			}
		case INDEX:
			fallthrough
		case KEY:
			err := setStmt(func() (interface{}, error) {
				indexStmt := CreateTableIndexStatement{}
				indexStmt.Kind = IndexKindNormal // TODO. separate to KEY and INDEX
				if err := p.parseColumnIndexKey(ctx, &indexStmt); err != nil {
					return nil, err
				}
				return &indexStmt, nil
			})
			if err != nil {
				return err
			}
		case FULLTEXT:
			err := setStmt(func() (interface{}, error) {
				indexStmt := CreateTableIndexStatement{}
				indexStmt.Kind = IndexKindFullText
				if err := p.parseColumnIndexFullTextKey(ctx, &indexStmt); err != nil {
					return nil, err
				}
				return &indexStmt, nil
			})
			if err != nil {
				return err
			}
		case SPARTIAL:
			err := setStmt(func() (interface{}, error) {
				indexStmt := CreateTableIndexStatement{}
				indexStmt.Kind = IndexKindSpartial
				if err := p.parseColumnIndexFullTextKey(ctx, &indexStmt); err != nil {
					return nil, err
				}
				return &indexStmt, nil
			})
			if err != nil {
				return err
			}
		case FOREIGN:
			err := setStmt(func() (interface{}, error) {
				indexStmt := CreateTableIndexStatement{}
				indexStmt.Kind = IndexKindForeignKey
				if err := p.parseColumnIndexForeignKey(ctx, &indexStmt); err != nil {
					return nil, err
				}
				return &indexStmt, nil
			})
			if err != nil {
				return err
			}
		case CHECK: // TODO
			return p.parseErrorf(ctx, "not support CHECK")
		case IDENT, BACKTICK_IDENT:

			err := setStmt(func() (interface{}, error) {
				colStmt := CreateTableColumnStatement{}
				colStmt.Name = t.Value

				var err error
				ctx.skipWhiteSpaces()
				switch t := ctx.next(); t.Type {
				case BIT:
					colStmt.Type = ColumnTypeBit
					err = p.parseColumnOption(ctx, &colStmt, ColumnOptionSize)
				case TINYINT:
					colStmt.Type = ColumnTypeTinyInt
					err = p.parseColumnOption(ctx, &colStmt, ColumnOptionFlagDigit)
				case SMALLINT:
					colStmt.Type = ColumnTypeSmallInt
					err = p.parseColumnOption(ctx, &colStmt, ColumnOptionFlagDigit)
				case MEDIUMINT:
					colStmt.Type = ColumnTypeMediumInt
					err = p.parseColumnOption(ctx, &colStmt, ColumnOptionFlagDigit)
				case INT:
					colStmt.Type = ColumnTypeInt
					err = p.parseColumnOption(ctx, &colStmt, ColumnOptionFlagDigit)
				case INTEGER:
					colStmt.Type = ColumnTypeInteger
					err = p.parseColumnOption(ctx, &colStmt, ColumnOptionFlagDigit)
				case BIGINT:
					colStmt.Type = ColumnTypeBigInt
					err = p.parseColumnOption(ctx, &colStmt, ColumnOptionFlagDigit)
				case REAL:
					colStmt.Type = ColumnTypeReal
					err = p.parseColumnOption(ctx, &colStmt, ColumnOptionFlagDecimal)
				case DOUBLE:
					colStmt.Type = ColumnTypeDouble
					err = p.parseColumnOption(ctx, &colStmt, ColumnOptionFlagDecimal)
				case FLOAT:
					colStmt.Type = ColumnTypeFloat
					err = p.parseColumnOption(ctx, &colStmt, ColumnOptionFlagDecimal)
				case DECIMAL:
					colStmt.Type = ColumnTypeDecimal
					err = p.parseColumnOption(ctx, &colStmt, ColumnOptionFlagDecimalOptional)
				case NUMERIC:
					colStmt.Type = ColumnTypeNumeric
					err = p.parseColumnOption(ctx, &colStmt, ColumnOptionFlagDecimalOptional)
				case DATE:
					colStmt.Type = ColumnTypeDate
					err = p.parseColumnOption(ctx, &colStmt, ColumnOptionFlagNone)
				case TIME:
					colStmt.Type = ColumnTypeTime
					err = p.parseColumnOption(ctx, &colStmt, ColumnOptionFlagTime)
				case TIMESTAMP:
					colStmt.Type = ColumnTypeTimestamp
					err = p.parseColumnOption(ctx, &colStmt, ColumnOptionFlagTime)
				case DATETIME:
					colStmt.Type = ColumnTypeDateTime
					err = p.parseColumnOption(ctx, &colStmt, ColumnOptionFlagTime)
				case YEAR:
					colStmt.Type = ColumnTypeYear
					err = p.parseColumnOption(ctx, &colStmt, ColumnOptionFlagNone)
				case CHAR:
					colStmt.Type = ColumnTypeChar
					err = p.parseColumnOption(ctx, &colStmt, ColumnOptionFlagChar)
				case VARCHAR:
					colStmt.Type = ColumnTypeVarChar
					err = p.parseColumnOption(ctx, &colStmt, ColumnOptionFlagChar)
				case BINARY:
					colStmt.Type = ColumnTypeBinary
					err = p.parseColumnOption(ctx, &colStmt, ColumnOptionFlagBinary)
				case VARBINARY:
					colStmt.Type = ColumnTypeVarBinary
					err = p.parseColumnOption(ctx, &colStmt, ColumnOptionFlagBinary)
				case TINYBLOB:
					colStmt.Type = ColumnTypeTinyBlob
					err = p.parseColumnOption(ctx, &colStmt, ColumnOptionFlagNone)
				case BLOB:
					colStmt.Type = ColumnTypeBlob
					err = p.parseColumnOption(ctx, &colStmt, ColumnOptionFlagNone)
				case MEDIUMBLOB:
					colStmt.Type = ColumnTypeMediumBlob
					err = p.parseColumnOption(ctx, &colStmt, ColumnOptionFlagNone)
				case LONGBLOB:
					colStmt.Type = ColumnTypeLongBlob
					err = p.parseColumnOption(ctx, &colStmt, ColumnOptionFlagNone)
				case TINYTEXT:
					colStmt.Type = ColumnTypeTinyText
					err = p.parseColumnOption(ctx, &colStmt, ColumnOptionFlagChar)
				case TEXT:
					colStmt.Type = ColumnTypeText
					err = p.parseColumnOption(ctx, &colStmt, ColumnOptionFlagChar)
				case MEDIUMTEXT:
					colStmt.Type = ColumnTypeMediumText
					err = p.parseColumnOption(ctx, &colStmt, ColumnOptionFlagChar)
				case LONGTEXT:
					colStmt.Type = ColumnTypeLongText
					err = p.parseColumnOption(ctx, &colStmt, ColumnOptionFlagChar)
				// case "ENUM":
				// case "SET":
				default:
					return nil, p.parseErrorf(ctx, "not supported type")
				}

				if err != nil {
					return nil, err
				}

				return &colStmt, nil
			})

			if err != nil {
				return err
			}
		default:
			return p.parseErrorf(ctx, "unexpected create table fields")
		}
	}
}

func (p *Parser) parseCreateTableOptions(ctx *parseCtx, stmt *CreateTableStatement) error {

	setOption := func(key string, types []TokenType) error {
		ctx.skipWhiteSpaces()
		if t := ctx.peek(); t.Type == EQUAL {
			ctx.advance()
			ctx.skipWhiteSpaces()
		}
		t := ctx.next()
		for _, typ := range types {
			if typ == t.Type {
				stmt.Options = append(stmt.Options, &CreateTableOptionStatement{key, t.Value})
				return nil
			}
		}
		return p.parseErrorf(ctx, "should %v", types)
	}

	for {
		ctx.skipWhiteSpaces()
		switch t := ctx.next(); t.Type {
		case ENGINE:
			if err := setOption("ENGINE", []TokenType{IDENT, BACKTICK_IDENT}); err != nil {
				return err
			}
		case AUTO_INCREMENT:
			if err := setOption("AUTO_INCREMENT", []TokenType{NUMBER}); err != nil {
				return err
			}
		case AVG_ROW_LENGTH:
			if err := setOption("AVG_ROW_LENGTH", []TokenType{NUMBER}); err != nil {
				return err
			}
		case DEFAULT:
			ctx.skipWhiteSpaces()
			switch t := ctx.next(); t.Type {
			case CHARACTER:
				ctx.skipWhiteSpaces()
				if t := ctx.next(); t.Type != SET {
					return p.parseErrorf(ctx, "expected SET")
				}
				if err := setOption("DEFAULT CHARACTER SET", []TokenType{IDENT, BACKTICK_IDENT}); err != nil {
					return err
				}
			case COLLATE:
				if err := setOption("DEFAULT COLLATE", []TokenType{IDENT, BACKTICK_IDENT}); err != nil {
					return err
				}
			default:
				return p.parseErrorf(ctx, "expected CHARACTER or COLLATE")
			}
		case CHARACTER:
			ctx.skipWhiteSpaces()
			if t := ctx.next(); t.Type != SET {
				return p.parseErrorf(ctx, "expected SET")
			}
			if err := setOption("DEFAULT CHARACTER SET", []TokenType{IDENT, BACKTICK_IDENT}); err != nil {
				return err
			}
		case COLLATE:
			if err := setOption("DEFAULT COLLATE", []TokenType{IDENT, BACKTICK_IDENT}); err != nil {
				return err
			}
		case CHECKSUM:
			if err := setOption("CHECKSUM", []TokenType{NUMBER}); err != nil {
				return err
			}
		case COMMENT:
			if err := setOption("COMMENT", []TokenType{SINGLE_QUOTE_IDENT, DOUBLE_QUOTE_IDENT}); err != nil {
				return err
			}
		case CONNECTION:
			if err := setOption("CONNECTION", []TokenType{SINGLE_QUOTE_IDENT, DOUBLE_QUOTE_IDENT}); err != nil {
				return err
			}
		case DATA:
			ctx.skipWhiteSpaces()
			if t := ctx.next(); t.Type != DIRECTORY {
				return p.parseErrorf(ctx, "should DIRECTORY")
			}
			if err := setOption("DATA DIRECTORY", []TokenType{SINGLE_QUOTE_IDENT, DOUBLE_QUOTE_IDENT}); err != nil {
				return err
			}
		case DELAY_KEY_WRITE:
			if err := setOption("DELAY_KEY_WRITE", []TokenType{NUMBER}); err != nil {
				return err
			}
		case INDEX:
			ctx.skipWhiteSpaces()
			if t := ctx.next(); t.Type != DIRECTORY {
				return p.parseErrorf(ctx, "should DIRECTORY")
			}
			if err := setOption("INDEX DIRECTORY", []TokenType{SINGLE_QUOTE_IDENT, DOUBLE_QUOTE_IDENT}); err != nil {
				return err
			}
		case INSERT_METHOD:
			if err := setOption("INSERT_METHOD", []TokenType{IDENT}); err != nil {
				return err
			}
		case KEY_BLOCK_SIZE:
			if err := setOption("KEY_BLOCK_SIZE", []TokenType{NUMBER}); err != nil {
				return err
			}
		case MAX_ROWS:
			if err := setOption("MAX_ROWS", []TokenType{NUMBER}); err != nil {
				return err
			}
		case MIN_ROWS:
			if err := setOption("MIN_ROWS", []TokenType{NUMBER}); err != nil {
				return err
			}
		case PACK_KEYS:
			if err := setOption("PACK_KEYS", []TokenType{NUMBER, IDENT}); err != nil {
				return err
			}
		case PASSWORD:
			if err := setOption("PASSWORD", []TokenType{SINGLE_QUOTE_IDENT, DOUBLE_QUOTE_IDENT}); err != nil {
				return err
			}
		case ROW_FORMAT:
			if err := setOption("ROW_FORMAT", []TokenType{DEFAULT, DYNAMIC, FIXED, COMPRESSED, REDUNDANT, COMPACT}); err != nil {
				return err
			}
		case STATS_AUTO_RECALC:
			if err := setOption("STATS_AUTO_RECALC", []TokenType{NUMBER, DEFAULT}); err != nil {
				return err
			}
		case STATS_PERSISTENT:
			if err := setOption("STATS_PERSISTENT", []TokenType{NUMBER, DEFAULT}); err != nil {
				return err
			}
		case STATS_SAMPLE_PAGES:
			if err := setOption("STATS_SAMPLE_PAGES", []TokenType{NUMBER}); err != nil {
				return err
			}
		case TABLESPACE:
			return p.parseErrorf(ctx, "not support TABLESPACE")
		case UNION:
			return p.parseErrorf(ctx, "not support UNION")
		case EOF:
			return nil
		case SEMICOLON:
			ctx.rewind()
			return nil
		default:
			return p.parseErrorf(ctx, "unexpected table options")
		}
	}
}

// parse for column
func (p *Parser) parseColumnOption(ctx *parseCtx, col *CreateTableColumnStatement, f int) error {
	f = f | ColumnOptionNull | ColumnOptionDefault | ColumnOptionAutoIncrement | ColumnOptionKey | ColumnOptionComment
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
			if check(ColumnOptionSize) {
				ctx.skipWhiteSpaces()
				t := ctx.next();
				if t.Type != NUMBER {
					return p.parseErrorf(ctx, "expected NUMBER (column size)")
				}
				tlen := t.Value

				ctx.skipWhiteSpaces()
				t = ctx.next()
				if t.Type != RPAREN {
					return p.parseErrorf(ctx, "expected RPAREN (column size)")
				}
				col.Length.Valid = true
				col.Length.Length = tlen
			} else if check(ColumnOptionDecimalSize) {
				strs, err := p.parseIdents(ctx, NUMBER, COMMA, NUMBER, RPAREN)
				if err != nil {
					return err
				}
				col.Length.Valid = true
				col.Length.Length = strs[0]
				col.Length.Decimals.Valid = true
				col.Length.Decimals.Value = strs[2]
			} else if check(ColumnOptionDecimalOptionalSize) {
				ctx.skipWhiteSpaces()
				t := ctx.next()
				if t.Type != NUMBER {
					return p.parseErrorf(ctx, "expected NUMBER (decimal size `M`)")
				}
				tlen := t.Value

				ctx.skipWhiteSpaces()
				t = ctx.next()
				if t.Type == RPAREN {
					col.Length.Valid = true
					col.Length.Length = tlen
					continue
				} else if t.Type != COMMA {
					return p.parseErrorf(ctx, "expected COMMA (decimal size)")
				}

				ctx.skipWhiteSpaces()
				t = ctx.next()
				if t.Type != NUMBER {
					return p.parseErrorf(ctx, "expected NUMBER (decimal size `D`)")
				}
				tscale := t.Value

				ctx.skipWhiteSpaces()
				if t := ctx.next(); t.Type != RPAREN {
					return p.parseErrorf(ctx, "expected RPARENT (decimal size)")
				}
				col.Length.Valid = true
				col.Length.Length = tlen
				col.Length.Decimals.Valid = true
				col.Length.Decimals.Value = tscale
			} else {
				return p.parseErrorf(ctx, "cant apply ColumnOptionSize, ColumnOptionDecimalSize, ColumnOptionDecimalOptionalSize")
			}
		case UNSIGNED:
			if !check(ColumnOptionUnsigned) {
				return p.parseErrorf(ctx, "cant apply UNSIGNED")
			}
			col.Unsgined = true
		case ZEROFILL:
			if !check(ColumnOptionZerofill) {
				return p.parseErrorf(ctx, "cant apply ZEROFILL")
			}
			col.ZeroFill = true
		case BINARY:
			if !check(ColumnOptionBinary) {
				return p.parseErrorf(ctx, "cant apply BINARY")
			}
			col.Binary = true
		case NOT:
			if !check(ColumnOptionNull) {
				return p.parseErrorf(ctx, "cant apply NOT NULL")
			}
			ctx.skipWhiteSpaces()
			switch t := ctx.next(); t.Type {
			case NULL:
				col.Null = ColumnOptionNullStateNotNull
			default:
				return p.parseErrorf(ctx, "should NULL")
			}
		case NULL:
			if !check(ColumnOptionNull) {
				return p.parseErrorf(ctx, "cant apply NULL")
			}
			col.Null = ColumnOptionNullStateNull
		case DEFAULT:
			if !check(ColumnOptionDefault) {
				return p.parseErrorf(ctx, "cant apply DEFAULT")
			}
			ctx.skipWhiteSpaces()
			switch t := ctx.next(); t.Type {
			case IDENT, SINGLE_QUOTE_IDENT, DOUBLE_QUOTE_IDENT, NUMBER, CURRENT_TIMESTAMP, NULL:
				col.Default.Valid = true
				col.Default.Value = t.Value
			default:
				return p.parseErrorf(ctx, "should IDENT, SINGLE_QUOTE_IDENT, DOUBLE_QUOTE_IDENT, NUMBER, CURRENT_TIMESTAMP, NULL")
			}
		case AUTO_INCREMENT:
			if !check(ColumnOptionAutoIncrement) {
				return p.parseErrorf(ctx, "cant apply AUTO_INCREMENT")
			}
			col.AutoIncrement = true
		case UNIQUE:
			if !check(ColumnOptionKey) {
				return p.parseErrorf(ctx, "cant apply UNIQUE KEY")
			}
			ctx.skipWhiteSpaces()
			if t := ctx.next(); t.Type == KEY {
				ctx.advance()
				col.Unique = true
			}
		case KEY:
			if !check(ColumnOptionKey) {
				return p.parseErrorf(ctx, "cant apply KEY")
			}
			col.Key = true
		case PRIMARY:
			if !check(ColumnOptionKey) {
				return p.parseErrorf(ctx, "cant apply PRIMARY KEY")
			}
			ctx.skipWhiteSpaces()
			if t := ctx.peek(); t.Type == KEY {
				ctx.advance()
				col.Primary = true
			}
		case COMMENT:
			if !check(ColumnOptionComment) {
				return p.parseErrorf(ctx, "cant apply COMMENT")
			}
			ctx.skipWhiteSpaces()
			switch t := ctx.next(); t.Type {
			case SINGLE_QUOTE_IDENT:
				col.Comment.Valid = true
				col.Comment.Value = t.Value
			default:
				return p.parseErrorf(ctx, "should SINGLE_QUOTE_IDENT")
			}
		case COMMA:
			ctx.rewind()
			return nil
		case RPAREN:
			ctx.rewind()
			return nil
		default:
			return p.parseErrorf(ctx, "unexpected column options")
		}
	}
}

func (p *Parser) parseColumnIndexPrimaryKey(ctx *parseCtx, stmt *CreateTableIndexStatement) error {
	ctx.skipWhiteSpaces()
	if t := ctx.next(); t.Type != KEY {
		return p.parseErrorf(ctx, "should KEY")
	}
	if err := p.parseColumnIndexType(ctx, stmt); err != nil {
		return err
	}

	cols, err := p.parseColumnIndexColName(ctx, stmt)
	if err != nil {
		return err
	}
	stmt.ColNames = append(stmt.ColNames, cols...)

	return nil
}

func (p *Parser) parseColumnIndexUniqueKey(ctx *parseCtx, stmt *CreateTableIndexStatement) error {
	ctx.skipWhiteSpaces()
	switch t := ctx.peek(); t.Type {
	case KEY, INDEX:
		ctx.advance()
	}

	if err := p.parseColumnIndexName(ctx, stmt); err != nil {
		return err
	}
	if err := p.parseColumnIndexType(ctx, stmt); err != nil {
		return err
	}

	cols, err := p.parseColumnIndexColName(ctx, stmt)
	if err != nil {
		return err
	}
	stmt.ColNames = append(stmt.ColNames, cols...)

	return nil
}

func (p *Parser) parseColumnIndexKey(ctx *parseCtx, stmt *CreateTableIndexStatement) error {
	if err := p.parseColumnIndexName(ctx, stmt); err != nil {
		return err
	}
	if err := p.parseColumnIndexType(ctx, stmt); err != nil {
		return err
	}

	cols, err := p.parseColumnIndexColName(ctx, stmt)
	if err != nil {
		return err
	}
	stmt.ColNames = append(stmt.ColNames, cols...)

	return nil
}

func (p *Parser) parseColumnIndexFullTextKey(ctx *parseCtx, stmt *CreateTableIndexStatement) error {
	if err := p.parseColumnIndexName(ctx, stmt); err != nil {
		return err
	}

	cols, err := p.parseColumnIndexColName(ctx, stmt)
	if err != nil {
		return err
	}
	stmt.ColNames = append(stmt.ColNames, cols...)

	return nil
}

func (p *Parser) parseColumnIndexForeignKey(ctx *parseCtx, stmt *CreateTableIndexStatement) error {
	ctx.skipWhiteSpaces()
	if t := ctx.next(); t.Type != KEY {
		return p.parseErrorf(ctx, "should KEY")
	}
	if err := p.parseColumnIndexName(ctx, stmt); err != nil {
		return err
	}

	cols, err := p.parseColumnIndexColName(ctx, stmt)
	if err != nil {
		return err
	}
	stmt.ColNames = append(stmt.ColNames, cols...)

	ctx.skipWhiteSpaces()
	if t := ctx.peek(); t.Type == REFERENCES {
		if err := p.parseColumnReference(ctx, stmt); err != nil {
			return err
		}
	}

	return nil
}

func (p *Parser) parseReferenceOption(ctx *parseCtx, opt *ReferenceOption) error {
	ctx.skipWhiteSpaces()
	switch t := ctx.next(); t.Type {
	case RESTRICT:
		*opt = ReferenceOptionRestrict
	case CASCADE:
		*opt = ReferenceOptionCascade
	case SET:
		ctx.skipWhiteSpaces()
		if t := ctx.next(); t.Type != NULL {
			return p.parseErrorf(ctx, "expected NULL")
		}
		*opt = ReferenceOptionSetNull
	case NO:
		ctx.skipWhiteSpaces()
		if t := ctx.next(); t.Type != ACTION {
			return p.parseErrorf(ctx, "expected ACTION")
		}
		*opt = ReferenceOptionNoAction
	default:
		return p.parseErrorf(ctx, "expected RESTRICT, CASCADE, SET or NO")
	}
	return nil
}

func (p *Parser) parseColumnReference(ctx *parseCtx, stmt *CreateTableIndexStatement) error {
	var r Reference

	ctx.skipWhiteSpaces()
	if t := ctx.next(); t.Type != REFERENCES {
		return p.parseErrorf(ctx, "expected REFERENCES")
	}

	ctx.skipWhiteSpaces()
	switch t := ctx.next(); t.Type {
	case BACKTICK_IDENT, IDENT:
		r.TableName = t.Value
	default:
		return p.parseErrorf(ctx, "should IDENT or BACKTICK_IDENT")
	}

	cols, err := p.parseColumnIndexColName(ctx, stmt)
	if err != nil {
		return err
	}
	r.ColNames = append(r.ColNames, cols...)

	ctx.skipWhiteSpaces()
	if t := ctx.peek(); t.Type == MATCH {
		ctx.advance()
		ctx.skipWhiteSpaces()
		switch t = ctx.next(); t.Type {
		case FULL:
			r.Match = ReferenceMatchFull
		case PARTIAL:
			r.Match = ReferenceMatchPartial
		case SIMPLE:
			r.Match = ReferenceMatchSimple
		default:
			return p.parseErrorf(ctx, "should FULL, PARTIAL or SIMPLE")
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
			if err := p.parseReferenceOption(ctx, &r.OnDelete); err != nil {
				return errors.Wrap(err, `failed to parse ON DELETE`)
			}
		case UPDATE:
			if err := p.parseReferenceOption(ctx, &r.OnUpdate); err != nil {
				return errors.Wrap(err, `failed to parse ON UPDATE`)
			}
			break OUTER
		default:
			return p.parseErrorf(ctx, "expected DELETE or UPDATE")
		}
	}

	stmt.Reference = &r
	return nil
}

func (p *Parser) parseColumnIndexName(ctx *parseCtx, stmt *CreateTableIndexStatement) error {
	ctx.skipWhiteSpaces()
	switch t := ctx.peek(); t.Type {
	case BACKTICK_IDENT, IDENT:
		ctx.advance()
		stmt.Name.Valid = true
		stmt.Name.Value = t.Value
	}
	return nil
}

func (p *Parser) parseColumnIndexTypeUsing(ctx *parseCtx, stmt *CreateTableIndexStatement) error {
	if t := ctx.next(); t.Type != USING {
		return errors.New(`expected USING`)
	}

	ctx.skipWhiteSpaces()
	switch t := ctx.next(); t.Type {
	case BTREE:
		stmt.Type = IndexTypeBtree
	case HASH:
		stmt.Type = IndexTypeHash
	default:
		return p.parseErrorf(ctx, "should BTREE or HASH")
	}
	return nil
}

func (p *Parser) parseColumnIndexType(ctx *parseCtx, stmt *CreateTableIndexStatement) error {
	ctx.skipWhiteSpaces()
	if t := ctx.peek(); t.Type == USING {
		return p.parseColumnIndexTypeUsing(ctx, stmt)
	}

	stmt.Type = IndexTypeNone
	return nil
}

// TODO rename method name
func (p *Parser) parseColumnIndexColName(ctx *parseCtx, stmt *CreateTableIndexStatement) ([]string, error) {
	var strs []string

	ctx.skipWhiteSpaces()
	if t := ctx.next(); t.Type != LPAREN {
		return nil, p.parseErrorf(ctx, "should (")
	}

	for {
		ctx.skipWhiteSpaces()
		t := ctx.next()
		if !(t.Type == IDENT || t.Type == BACKTICK_IDENT) {
			return nil, p.parseErrorf(ctx, "should IDENT or BACKTICK_IDENT")
		}
		strs = append(strs, t.Value)

		ctx.skipWhiteSpaces()
		switch t = ctx.next(); t.Type {
		case COMMA:
			// search next
			continue
		case RPAREN:
			return strs, nil
		default:
			return nil, p.parseErrorf(ctx, "should , or )")
		}
	}
}

// Skips over whitespaces. Once this method returns, you can be
// certain that next call to ctx.next()/peek() will result in a
// non-space token
func (ctx *parseCtx) skipWhiteSpaces() {
	for {
		switch t := ctx.peek(); t.Type {
		case SPACE, COMMENT_IDENT:
			ctx.advance()
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
			return nil, p.parseErrorf(ctx, "should %v", idents)
		}
		strs = append(strs, t.Value)
	}
	return strs, nil
}

func (p *Parser) eol(ctx *parseCtx) bool {
	ctx.skipWhiteSpaces()
	switch t := ctx.peek(); t.Type {
	case EOF, SEMICOLON:
		ctx.advance()
		return true
	default:
		return false
	}
}

func (p *Parser) parseErrorf(ctx *parseCtx, format string, a ...interface{}) error {
	pos1 := int(math.Max(float64(ctx.lexer.pos-ctx.errorContext), 0))
	pos2 := int(math.Min(float64(ctx.lexer.pos+ctx.errorContext), float64(len(ctx.lexer.input))))
	return fmt.Errorf("parse error:%s pos: %s%s%s", fmt.Sprintf(format, a...), ctx.lexer.input[pos1:ctx.lexer.pos], ctx.errorMarker, ctx.lexer.input[ctx.lexer.pos:pos2])
}
