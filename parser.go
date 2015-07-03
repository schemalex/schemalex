package schemalex

import (
	"fmt"
	"math"
)

type Parser struct {
	lexer        *lexer
	ErrorMarker  string
	ErrorContext int
}

func NewParser(str string) *Parser {
	return &Parser{
		lexer: &lexer{
			input: str,
		},
		ErrorMarker:  "___",
		ErrorContext: 20,
	}
}

func (p *Parser) Parse() ([]Stmt, error) {
	stmts := []Stmt{}
LOOP:
	for {
		t, _ := p.parseIgnoreWhiteSpace()
	S1:
		switch t {
		case CREATE:
			t, _ := p.parseIgnoreWhiteSpace()
			switch t {
			case DATABASE:
				_, err := p.parseCreateDatabase()
				if err != nil {
					return nil, err
				}
				// stmts = append(stmts, stmt)
				break S1
			case TABLE:
				stmt, err := p.parseCreateTable()
				if err != nil {
					return nil, err
				}
				stmts = append(stmts, stmt)
				break S1
			default:
				return nil, p.parseErrorf("should DATABASE or TABLE")
			}
		case COMMENT_IDENT:
		case DROP, SET, USE:
			for {
				if p.eol() {
					break S1
				}
			}
		case EOF:
			break LOOP
		default:
			return nil, p.parseErrorf("should CREATE, COMMENT_IDENT or EOF")
		}
	}

	return stmts, nil
}

// https://dev.mysql.com/doc/refman/5.5/en/create-database.html
// TODO: charset, collation
func (p *Parser) parseCreateDatabase() (*CreateDatabaseStatement, error) {
	stmt := &CreateDatabaseStatement{}
	t, str := p.parseIgnoreWhiteSpace()
	setname := func() error {
		switch t {
		case IDENT, BACKTICK_IDENT:
			stmt.Name = str
		default:
			return p.parseErrorf("should IDENT or BACKTICK_IDENT")
		}
		if p.eol() {
			return nil
		} else {
			return p.parseErrorf("should EOL")
		}
	}
	switch t {
	case IDENT, BACKTICK_IDENT:
		if err := setname(); err != nil {
			return nil, err
		}
		return stmt, nil
	case IF:
		if _, err := p.parseIndents([]token{NOT, EXISTS}); err != nil {
			return nil, err
		}
		t, str = p.parseIgnoreWhiteSpace()
		stmt.IfNotExist = true
		if err := setname(); err != nil {
			return nil, err
		}
		return stmt, nil
	default:
		return nil, p.parseErrorf("should IDENT, BACKTICK_IDENT or IF")
	}
}

// http://dev.mysql.com/doc/refman/5.6/en/create-table.html
func (p *Parser) parseCreateTable() (*CreateTableStatement, error) {
	stmt := CreateTableStatement{}
	t, str := p.parseIgnoreWhiteSpace()

	switch t {
	case TEMPORARY:
		stmt.Temporary = true
		t, str = p.parseIgnoreWhiteSpace()
		if !(t == IDENT || t == BACKTICK_IDENT) {
			return nil, p.parseErrorf("should IDENT or BACKTICK_IDENT")
		}
		fallthrough
	case IDENT, BACKTICK_IDENT:

		stmt.Name = str
		t, _ := p.parseIgnoreWhiteSpace()

		if t == IF {
			if _, err := p.parseIndents([]token{NOT, EXISTS}); err != nil {
				return nil, p.parseErrorf("should NOT EXISTS")
			}
			stmt.IfNotExist = true
			t, _ = p.parseIgnoreWhiteSpace()
		}

		if t != LPAREN {
			return nil, p.parseErrorf("should (")
		}

		if err := p.parseCreateTableFields(&stmt); err != nil {
			return nil, err
		}

		return &stmt, nil
	default:
		return nil, p.parseErrorf("should TEMPORARY, IDENT or BACKTICK_IDENT")
	}
}

func (p *Parser) parseCreateTableFields(stmt *CreateTableStatement) error {
	var targetStmt interface{}

	appendStmt := func() {
		switch t := targetStmt.(type) {
		case CreateTableIndexStatement:
			stmt.Indexes = append(stmt.Indexes, t)
		case CreateTableColumnStatement:
			stmt.Columns = append(stmt.Columns, t)
		default:
			panic("not reach")
		}
		targetStmt = nil
	}

	setStmt := func(f func() (interface{}, error)) error {
		if targetStmt != nil {
			return p.parseErrorf("seems not to be end previous column or index definition")
		}
		stmt, err := f()
		if err != nil {
			return err
		}
		targetStmt = stmt
		return nil
	}

	for {
		t, str := p.parseIgnoreWhiteSpace()
		switch t {
		case RPAREN:
			appendStmt()
			if err := p.parseCreateTableOptions(stmt); err != nil {
				return err
			}
			// partition option
			if !p.eol() {
				return p.parseErrorf("should EOL")
			}
			return nil
		case COMMA:
			if targetStmt == nil {
				return p.parseErrorf("unexpected COMMA")
			}
			appendStmt()
		case CONSTRAINT:
			err := setStmt(func() (interface{}, error) {
				indexStmt := CreateTableIndexStatement{}
				t, str := p.parseIgnoreWhiteSpace()
				if t == IDENT || t == BACKTICK_IDENT {
					// TODO: should smart
					copyStr := str
					indexStmt.Symbol = &copyStr
					t, str = p.parseIgnoreWhiteSpace()
				}

				switch t {
				case PRIMARY:
					indexStmt.Kind = IndexKindPrimaryKey
					if err := p.parseColumnIndexPrimaryKey(&indexStmt); err != nil {
						return nil, err
					}
				case UNIQUE:
					indexStmt.Kind = IndexKindUnique
					if err := p.parseColumnIndexUniqueKey(&indexStmt); err != nil {
						return nil, err
					}
				case FOREIGN:
					indexStmt.Kind = IndexKindForeignKey
					if err := p.parseColumnIndexForeignKey(&indexStmt); err != nil {
						return nil, err
					}
				default:
					return nil, p.parseErrorf("not supported")
				}
				return indexStmt, nil
			})
			if err != nil {
				return err
			}
		case PRIMARY:
			err := setStmt(func() (interface{}, error) {
				indexStmt := CreateTableIndexStatement{}
				indexStmt.Kind = IndexKindPrimaryKey
				if err := p.parseColumnIndexPrimaryKey(&indexStmt); err != nil {
					return nil, err
				}
				return indexStmt, nil
			})
			if err != nil {
				return err
			}
		case UNIQUE:
			err := setStmt(func() (interface{}, error) {
				indexStmt := CreateTableIndexStatement{}
				indexStmt.Kind = IndexKindUnique
				if err := p.parseColumnIndexUniqueKey(&indexStmt); err != nil {
					return nil, err
				}
				return indexStmt, nil
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
				if err := p.parseColumnIndexKey(&indexStmt); err != nil {
					return nil, err
				}
				return indexStmt, nil
			})
			if err != nil {
				return err
			}
		case FULLTEXT:
			err := setStmt(func() (interface{}, error) {
				indexStmt := CreateTableIndexStatement{}
				indexStmt.Kind = IndexKindFullText
				if err := p.parseColumnIndexFullTextKey(&indexStmt); err != nil {
					return nil, err
				}
				return indexStmt, nil
			})
			if err != nil {
				return err
			}
		case SPARTIAL:
			err := setStmt(func() (interface{}, error) {
				indexStmt := CreateTableIndexStatement{}
				indexStmt.Kind = IndexKindSpartial
				if err := p.parseColumnIndexFullTextKey(&indexStmt); err != nil {
					return nil, err
				}
				return indexStmt, nil
			})
			if err != nil {
				return err
			}
		case FOREIGN:
			err := setStmt(func() (interface{}, error) {
				indexStmt := CreateTableIndexStatement{}
				indexStmt.Kind = IndexKindForeignKey
				if err := p.parseColumnIndexForeignKey(&indexStmt); err != nil {
					return nil, err
				}
				return indexStmt, nil
			})
			if err != nil {
				return err
			}
		case CHECK: // TODO
			return p.parseErrorf("not support CHECK")
		case IDENT, BACKTICK_IDENT:

			err := setStmt(func() (interface{}, error) {
				colStmt := CreateTableColumnStatement{}
				colStmt.Name = str
				t, _ := p.parseIgnoreWhiteSpace()

				var err error
				switch t {
				case BIT:
					colStmt.Type = ColumnTypeBit
					err = p.parseColumnOption(&colStmt, ColumnOptionSize)
				case TINYINT:
					colStmt.Type = ColumnTypeTinyInt
					err = p.parseColumnOption(&colStmt, ColumnOptionFlagDigit)
				case SMALLINT:
					colStmt.Type = ColumnTypeSmallInt
					err = p.parseColumnOption(&colStmt, ColumnOptionFlagDigit)
				case MEDIUMINT:
					colStmt.Type = ColumnTypeMediumInt
					err = p.parseColumnOption(&colStmt, ColumnOptionFlagDigit)
				case INT:
					colStmt.Type = ColumnTypeInt
					err = p.parseColumnOption(&colStmt, ColumnOptionFlagDigit)
				case INTEGER:
					colStmt.Type = ColumnTypeInteger
					err = p.parseColumnOption(&colStmt, ColumnOptionFlagDigit)
				case BIGINT:
					colStmt.Type = ColumnTypeBigInt
					err = p.parseColumnOption(&colStmt, ColumnOptionFlagDigit)
				case REAL:
					colStmt.Type = ColumnTypeReal
					err = p.parseColumnOption(&colStmt, ColumnOptionFlagDecimal)
				case DOUBLE:
					colStmt.Type = ColumnTypeDouble
					err = p.parseColumnOption(&colStmt, ColumnOptionFlagDecimal)
				case FLOAT:
					colStmt.Type = ColumnTypeFloat
					err = p.parseColumnOption(&colStmt, ColumnOptionFlagDecimal)
				case DECIMAL:
					colStmt.Type = ColumnTypeDecimal
					err = p.parseColumnOption(&colStmt, ColumnOptionFlagDecimalOptional)
				case NUMERIC:
					colStmt.Type = ColumnTypeNumeric
					err = p.parseColumnOption(&colStmt, ColumnOptionFlagDecimalOptional)
				case DATE:
					colStmt.Type = ColumnTypeDate
					err = p.parseColumnOption(&colStmt, ColumnOptionFlagNone)
				case TIME:
					colStmt.Type = ColumnTypeTime
					err = p.parseColumnOption(&colStmt, ColumnOptionFlagTime)
				case TIMESTAMP:
					colStmt.Type = ColumnTypeTimestamp
					err = p.parseColumnOption(&colStmt, ColumnOptionFlagTime)
				case DATETIME:
					colStmt.Type = ColumnTypeDateTime
					err = p.parseColumnOption(&colStmt, ColumnOptionFlagTime)
				case YEAR:
					colStmt.Type = ColumnTypeYear
					err = p.parseColumnOption(&colStmt, ColumnOptionFlagNone)
				case CHAR:
					colStmt.Type = ColumnTypeChar
					err = p.parseColumnOption(&colStmt, ColumnOptionFlagChar)
				case VARCHAR:
					colStmt.Type = ColumnTypeVarChar
					err = p.parseColumnOption(&colStmt, ColumnOptionFlagChar)
				case BINARY:
					colStmt.Type = ColumnTypeBinary
					err = p.parseColumnOption(&colStmt, ColumnOptionFlagBinary)
				case VARBINARY:
					colStmt.Type = ColumnTypeVarBinary
					err = p.parseColumnOption(&colStmt, ColumnOptionFlagBinary)
				case TINYBLOB:
					colStmt.Type = ColumnTypeTinyBlob
					err = p.parseColumnOption(&colStmt, ColumnOptionFlagNone)
				case BLOB:
					colStmt.Type = ColumnTypeBlob
					err = p.parseColumnOption(&colStmt, ColumnOptionFlagNone)
				case MEDIUMBLOB:
					colStmt.Type = ColumnTypeMediumBlob
					err = p.parseColumnOption(&colStmt, ColumnOptionFlagNone)
				case LONGBLOB:
					colStmt.Type = ColumnTypeLongBlob
					err = p.parseColumnOption(&colStmt, ColumnOptionFlagNone)
				case TINYTEXT:
					colStmt.Type = ColumnTypeTinyText
					err = p.parseColumnOption(&colStmt, ColumnOptionFlagChar)
				case TEXT:
					colStmt.Type = ColumnTypeText
					err = p.parseColumnOption(&colStmt, ColumnOptionFlagChar)
				case MEDIUMTEXT:
					colStmt.Type = ColumnTypeMediumText
					err = p.parseColumnOption(&colStmt, ColumnOptionFlagChar)
				case LONGTEXT:
					colStmt.Type = ColumnTypeLongText
					err = p.parseColumnOption(&colStmt, ColumnOptionFlagChar)
				// case "ENUM":
				// case "SET":
				default:
					return nil, p.parseErrorf("not supported type")
				}

				if err != nil {
					return nil, err
				}

				return colStmt, nil
			})

			if err != nil {
				return err
			}
		default:
			return p.parseErrorf("unexpected create table fields")
		}
	}
}

func (p *Parser) parseCreateTableOptions(stmt *CreateTableStatement) error {

	setOption := func(key string, tokens []token) error {
		t, str := p.parseIgnoreWhiteSpace()
		if t == EQUAL {
			t, str = p.parseIgnoreWhiteSpace()
		}
		for _, _token := range tokens {
			if _token == t {
				stmt.Options = append(stmt.Options, CreateTableOptionStatement{key, str})
				return nil
			}
		}
		return p.parseErrorf("should %v", tokens)
	}

	for {
		t, _ := p.parseIgnoreWhiteSpace()
		switch t {
		case ENGINE:
			if err := setOption("ENGINE", []token{IDENT, BACKTICK_IDENT}); err != nil {
				return err
			}
		case AUTO_INCREMENT:
			if err := setOption("AUTO_INCREMENT", []token{NUMBER}); err != nil {
				return err
			}
		case AVG_ROW_LENGTH:
			if err := setOption("AVG_ROW_LENGTH", []token{NUMBER}); err != nil {
				return err
			}
		case DEFAULT:
			t, _ := p.parseIgnoreWhiteSpace()
			switch t {
			case CHARACTER:
				t, _ := p.parseIgnoreWhiteSpace()
				if t != SET {
					return p.parseErrorf("should SET")
				}
				if err := setOption("DEFAULT CHARACTER SET", []token{IDENT, BACKTICK_IDENT}); err != nil {
					return err
				}
			case COLLATE:
				if err := setOption("DEFAULT COLLATE", []token{IDENT, BACKTICK_IDENT}); err != nil {
					return err
				}
			default:
				return p.parseErrorf("should CHARACTER or COLLATE")
			}
		case CHARACTER:
			t, _ := p.parseIgnoreWhiteSpace()
			if t != SET {
				return p.parseErrorf("should SET")
			}
			if err := setOption("DEFAULT CHARACTER SET", []token{IDENT, BACKTICK_IDENT}); err != nil {
				return err
			}
		case COLLATE:
			if err := setOption("DEFAULT COLLATE", []token{IDENT, BACKTICK_IDENT}); err != nil {
				return err
			}
		case CHECKSUM:
			if err := setOption("CHECKSUM", []token{NUMBER}); err != nil {
				return err
			}
		case COMMENT:
			if err := setOption("COMMENT", []token{SINGLE_QUOTE_IDENT, DOUBLE_QUOTE_IDENT}); err != nil {
				return err
			}
		case CONNECTION:
			if err := setOption("CONNECTION", []token{SINGLE_QUOTE_IDENT, DOUBLE_QUOTE_IDENT}); err != nil {
				return err
			}
		case DATA:
			t, _ := p.parseIgnoreWhiteSpace()
			if t != DIRECTORY {
				return p.parseErrorf("should DIRECTORY")
			}
			if err := setOption("DATA DIRECTORY", []token{SINGLE_QUOTE_IDENT, DOUBLE_QUOTE_IDENT}); err != nil {
				return err
			}
		case DELAY_KEY_WRITE:
			if err := setOption("DELAY_KEY_WRITE", []token{NUMBER}); err != nil {
				return err
			}
		case INDEX:
			t, _ := p.parseIgnoreWhiteSpace()
			if t != DIRECTORY {
				return p.parseErrorf("should DIRECTORY")
			}
			if err := setOption("INDEX DIRECTORY", []token{SINGLE_QUOTE_IDENT, DOUBLE_QUOTE_IDENT}); err != nil {
				return err
			}
		case INSERT_METHOD:
			if err := setOption("INSERT_METHOD", []token{IDENT}); err != nil {
				return err
			}
		case KEY_BLOCK_SIZE:
			if err := setOption("KEY_BLOCK_SIZE", []token{NUMBER}); err != nil {
				return err
			}
		case MAX_ROWS:
			if err := setOption("MAX_ROWS", []token{NUMBER}); err != nil {
				return err
			}
		case MIN_ROWS:
			if err := setOption("MIN_ROWS", []token{NUMBER}); err != nil {
				return err
			}
		case PACK_KEYS:
			if err := setOption("PACK_KEYS", []token{NUMBER, IDENT}); err != nil {
				return err
			}
		case PASSWORD:
			if err := setOption("PASSWORD", []token{SINGLE_QUOTE_IDENT, DOUBLE_QUOTE_IDENT}); err != nil {
				return err
			}
		case ROW_FORMAT:
			if err := setOption("ROW_FORMAT", []token{DEFAULT, DYNAMIC, FIXED, COMPRESSED, REDUNDANT, COMPACT}); err != nil {
				return err
			}
		case STATS_AUTO_RECALC:
			if err := setOption("STATS_AUTO_RECALC", []token{NUMBER, DEFAULT}); err != nil {
				return err
			}
		case STATS_PERSISTENT:
			if err := setOption("STATS_PERSISTENT", []token{NUMBER, DEFAULT}); err != nil {
				return err
			}
		case STATS_SAMPLE_PAGES:
			if err := setOption("STATS_SAMPLE_PAGES", []token{NUMBER}); err != nil {
				return err
			}
		case TABLESPACE:
			return p.parseErrorf("not support TABLESPACE")
		case UNION:
			return p.parseErrorf("not support UNION")
		case EOF:
			return nil
		case SEMICORON:
			p.reset()
			return nil
		default:
			return p.parseErrorf("unexpected table options")
		}
	}
}

// parse for column
func (p *Parser) parseColumnOption(col *CreateTableColumnStatement, f int) error {
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
		t, _ := p.parseIgnoreWhiteSpace()
		switch t {
		case LPAREN:
			if check(ColumnOptionSize) {
				t, length := p.parseIgnoreWhiteSpace()
				if t != NUMBER {
					return p.parseErrorf("should NUMBER")
				}
				t, _ = p.parseIgnoreWhiteSpace()
				if t != RPAREN {
					return p.parseErrorf("should )")
				}
				col.Length = &LengthNumber{length}
			} else if check(ColumnOptionDecimalSize) {
				strs, err := p.parseIndents([]token{NUMBER, COMMA, NUMBER, RPAREN})
				if err != nil {
					return err
				}
				col.Length = &LengthDecimal{strs[0], strs[2]}
			} else if check(ColumnOptionDecimalOptionalSize) {
				t, length := p.parseIgnoreWhiteSpace()
				if t != NUMBER {
					return p.parseErrorf("should NUMBER")
				}
				t, _ = p.parseIgnoreWhiteSpace()
				if t == RPAREN {
					col.Length = LengthOptionalDecimal{length, nil}
					continue
				} else if t != COMMA {
					return p.parseErrorf("should ,")
				}
				t, decimal := p.parseIgnoreWhiteSpace()
				if t != NUMBER {
					return p.parseErrorf("should NUMBER")
				}
				t, _ = p.parseIgnoreWhiteSpace()
				if t != RPAREN {
					return p.parseErrorf("should )")
				}
				col.Length = LengthOptionalDecimal{length, &decimal}
			} else {
				return p.parseErrorf("cant apply ColumnOptionSize, ColumnOptionDecimalSize, ColumnOptionDecimalOptionalSize")
			}
		case UNSIGNED:
			if !check(ColumnOptionUnsigned) {
				return p.parseErrorf("cant apply UNSIGNED")
			}
			col.Unsgined = true
		case ZEROFILL:
			if !check(ColumnOptionZerofill) {
				return p.parseErrorf("cant apply ZEROFILL")
			}
			col.ZeroFill = true
		case BINARY:
			if !check(ColumnOptionBinary) {
				return p.parseErrorf("cant apply BINARY")
			}
			col.Binary = true
		case NOT:
			if !check(ColumnOptionNull) {
				return p.parseErrorf("cant apply NOT NULL")
			}
			t, _ := p.parseIgnoreWhiteSpace()
			if t == NULL {
				col.Null = ColumnOptionNullStateNotNull
			} else {
				return p.parseErrorf("should NULL")
			}
		case NULL:
			if !check(ColumnOptionNull) {
				return p.parseErrorf("cant apply NULL")
			}
			col.Null = ColumnOptionNullStateNull
		case DEFAULT:
			if !check(ColumnOptionDefault) {
				return p.parseErrorf("cant apply DEFAULT")
			}
			// TODO type
			t, str := p.parseIgnoreWhiteSpace()
			switch t {
			case IDENT, SINGLE_QUOTE_IDENT, DOUBLE_QUOTE_IDENT, NUMBER, CURRENT_TIMESTAMP, NULL:
				col.Default = &str
			default:
				return p.parseErrorf("should IDENT, SINGLE_QUOTE_IDENT, DOUBLE_QUOTE_IDENT, NUMBER, CURRENT_TIMESTAMP, NULL")
			}
		case AUTO_INCREMENT:
			if !check(ColumnOptionAutoIncrement) {
				return p.parseErrorf("cant apply AUTO_INCREMENT")
			}
			col.AutoIncrement = true
		case UNIQUE:
			if !check(ColumnOptionKey) {
				return p.parseErrorf("cant apply UNIQUE KEY")
			}
			t, _ := p.parseIgnoreWhiteSpace()
			if t != KEY {
				p.reset()
			}
			col.Unique = true
		case KEY:
			if !check(ColumnOptionKey) {
				return p.parseErrorf("cant apply KEY")
			}
			col.Key = true
		case PRIMARY:
			if !check(ColumnOptionKey) {
				return p.parseErrorf("cant apply PRIMARY KEY")
			}
			t, _ := p.parseIgnoreWhiteSpace()
			if t != KEY {
				p.reset()
			}
			col.Primary = true
		case COMMENT:
			if !check(ColumnOptionComment) {
				return p.parseErrorf("cant apply COMMENT")
			}
			t, str := p.parseIgnoreWhiteSpace()
			if t != SINGLE_QUOTE_IDENT {
				return p.parseErrorf("should SINGLE_QUOTE_IDENT")
			}
			col.Comment = &str
		case COMMA:
			p.reset()
			return nil
		case RPAREN:
			p.reset()
			return nil
		default:
			return p.parseErrorf("unexpected column options")
		}
	}
}

func (p *Parser) parseColumnIndexPrimaryKey(stmt *CreateTableIndexStatement) error {
	t, _ := p.parseIgnoreWhiteSpace()
	if t != KEY {
		return p.parseErrorf("should KEY")
	}
	if err := p.parseColumnIndexType(stmt); err != nil {
		return err
	}

	cols, err := p.parseColumnIndexColName(stmt)
	if err != nil {
		return err
	}
	stmt.ColNames = append(stmt.ColNames, cols...)

	return nil
}

func (p *Parser) parseColumnIndexUniqueKey(stmt *CreateTableIndexStatement) error {
	t, _ := p.parseIgnoreWhiteSpace()
	if !(t == KEY || t == INDEX) {
		p.reset()
	}

	if err := p.parseColumnIndexName(stmt); err != nil {
		return err
	}
	if err := p.parseColumnIndexType(stmt); err != nil {
		return err
	}

	cols, err := p.parseColumnIndexColName(stmt)
	if err != nil {
		return err
	}
	stmt.ColNames = append(stmt.ColNames, cols...)

	return nil
}

func (p *Parser) parseColumnIndexKey(stmt *CreateTableIndexStatement) error {
	if err := p.parseColumnIndexName(stmt); err != nil {
		return err
	}
	if err := p.parseColumnIndexType(stmt); err != nil {
		return err
	}

	cols, err := p.parseColumnIndexColName(stmt)
	if err != nil {
		return err
	}
	stmt.ColNames = append(stmt.ColNames, cols...)

	return nil
}

func (p *Parser) parseColumnIndexFullTextKey(stmt *CreateTableIndexStatement) error {
	if err := p.parseColumnIndexName(stmt); err != nil {
		return err
	}

	cols, err := p.parseColumnIndexColName(stmt)
	if err != nil {
		return err
	}
	stmt.ColNames = append(stmt.ColNames, cols...)

	return nil
}

func (p *Parser) parseColumnIndexForeignKey(stmt *CreateTableIndexStatement) error {
	t, _ := p.parseIgnoreWhiteSpace()
	if t != KEY {
		return p.parseErrorf("should KEY")
	}
	if err := p.parseColumnIndexName(stmt); err != nil {
		return err
	}

	cols, err := p.parseColumnIndexColName(stmt)
	if err != nil {
		return err
	}
	stmt.ColNames = append(stmt.ColNames, cols...)

	t, _ = p.parseIgnoreWhiteSpace()
	p.reset()
	if t == REFERENCES {
		if err := p.parseColumnReference(stmt); err != nil {
			return err
		}
	}

	return nil
}

func (p *Parser) parseColumnReference(stmt *CreateTableIndexStatement) error {
	var r Reference

	t, _ := p.parseIgnoreWhiteSpace()
	if t != REFERENCES {
		return p.parseErrorf("should REFERENCES")
	}

	t, tableName := p.parseIgnoreWhiteSpace()
	if !(t == IDENT || t == BACKTICK_IDENT) {
		return p.parseErrorf("should IDENT or BACKTICK_IDENT")
	}
	r.TableName = tableName

	cols, err := p.parseColumnIndexColName(stmt)
	if err != nil {
		return err
	}
	r.ColNames = append(r.ColNames, cols...)

	t, _ = p.parseIgnoreWhiteSpace()
	if t == MATCH {
		t, _ := p.parseIgnoreWhiteSpace()
		switch t {
		case FULL:
			r.Match = ReferenceMatchFull
		case PARTIAL:
			r.Match = ReferenceMatchPartial
		case SIMPLE:
			r.Match = ReferenceMatchSimple
		default:
			return p.parseErrorf("should FULL, PARTIAL or SIMPLE")
		}
		t, _ = p.parseIgnoreWhiteSpace()
	}

	if t != ON {
		p.reset()
		stmt.Reference = &r
		return nil
	}

	parseRefenceOption := func() (ReferenceOption, error) {
		t, _ = p.parseIgnoreWhiteSpace()
		switch t {
		case RESTRICT:
			return ReferenceOptionRestrict, nil
		case CASCADE:
			return ReferenceOptionCascade, nil
		case SET:
			t, _ := p.parseIgnoreWhiteSpace()
			if t != NULL {
				return 0, p.parseErrorf("should NULL")
			}
			return ReferenceOptionSetNull, nil
		case NO:
			t, _ := p.parseIgnoreWhiteSpace()
			if t != ACTION {
				return 0, p.parseErrorf("should ACTION")
			}
			return ReferenceOptionNoAction, nil
		default:
			return 0, p.parseErrorf("should RESTRICT, CASCADE, SET or NO")
		}
	}

	t, _ = p.parseIgnoreWhiteSpace()
	switch t {
	case DELETE:
		option, err := parseRefenceOption()
		if err != nil {
			return err
		}
		r.OnDelete = option
	case UPDATE:
		option, err := parseRefenceOption()
		if err != nil {
			return err
		}
		r.OnUpdate = option
		stmt.Reference = &r
		return nil
	default:
		return p.parseErrorf("should DELETE or UPDATE")
	}

	t, _ = p.parseIgnoreWhiteSpace()
	if t != ON {
		p.reset()
		stmt.Reference = &r
		return nil
	}

	t, _ = p.parseIgnoreWhiteSpace()
	switch t {
	case UPDATE:
		option, err := parseRefenceOption()
		if err != nil {
			return err
		}
		r.OnUpdate = option
	default:
		return p.parseErrorf("should UPDATE")
	}

	stmt.Reference = &r

	return nil
}

func (p *Parser) parseColumnIndexName(stmt *CreateTableIndexStatement) error {
	t, s := p.parseIgnoreWhiteSpace()
	if t == BACKTICK_IDENT || t == IDENT {
		stmt.Name = &s
	} else {
		p.reset()
	}
	return nil
}

func (p *Parser) parseColumnIndexType(stmt *CreateTableIndexStatement) error {
	t, _ := p.parseIgnoreWhiteSpace()
	if t == USING {
		t, _ = p.parseIgnoreWhiteSpace()
		switch t {
		case BTREE:
			stmt.Type = IndexTypeBtree
		case HASH:
			stmt.Type = IndexTypeHash
		default:
			return p.parseErrorf("should BTREE or HASH")
		}
	} else {
		p.reset()
		stmt.Type = IndexTypeNone
	}
	return nil
}

// TODO rename method name
func (p *Parser) parseColumnIndexColName(stmt *CreateTableIndexStatement) ([]string, error) {
	var strs []string

	t, _ := p.parseIgnoreWhiteSpace()
	if t != LPAREN {
		return nil, p.parseErrorf("should (")
	}

	for {
		t, s := p.parseIgnoreWhiteSpace()
		if !(t == IDENT || t == BACKTICK_IDENT) {
			return nil, p.parseErrorf("should IDENT or BACKTICK_IDENT")
		}
		strs = append(strs, s)
		t, s = p.parseIgnoreWhiteSpace()
		switch t {
		case COMMA:
			// search next
			continue
		case RPAREN:
			return strs, nil
		default:
			return nil, p.parseErrorf("should , or )")
		}
	}
}

// util
func (p *Parser) parseIgnoreWhiteSpace() (token, string) {
	for {
		t, i := p.lexer.read()
		//log.Println("parseIgnoreWhiteSpace:", int(t), p.lexer.str())

		if t == SPACE || t == COMMENT_IDENT {
			continue
		}

		return t, i
	}

	return ILLEAGAL, ""
}

func (p *Parser) parseIndents(idents []token) ([]string, error) {
	strs := []string{}
	for _, ident := range idents {
		t, str := p.parseIgnoreWhiteSpace()
		if t != ident {
			return nil, p.parseErrorf("should %v", idents)
		}
		strs = append(strs, str)
	}
	return strs, nil
}

func (p *Parser) eol() bool {
	t, _ := p.parseIgnoreWhiteSpace()
	switch t {
	case EOF, SEMICORON:
		return true
	default:
		return false
	}
}

func (p *Parser) reset() {
	p.lexer.pos = p.lexer.start
}

func (p *Parser) parseErrorf(format string, a ...interface{}) error {
	pos1 := int(math.Max(float64(p.lexer.pos-p.ErrorContext), 0))
	pos2 := int(math.Min(float64(p.lexer.pos+p.ErrorContext), float64(len(p.lexer.input))))
	return fmt.Errorf("parse error:%s pos: %s%s%s", fmt.Sprintf(format, a...), p.lexer.input[pos1:p.lexer.pos], p.ErrorMarker, p.lexer.input[p.lexer.pos:pos2])
}
