package schemalex

import (
	"io"

	"github.com/schemalex/schemalex/statement"
)

type Stmt interface {
	WriteTo(io.Writer) (int64, error)
}

type Statements []Stmt

type CreateDatabaseStatement struct {
	Name       string
	IfNotExist bool
}

type CreateTableStatement struct {
	Name       string
	Temporary  bool
	IfNotExist bool
	Columns    []statement.TableColumn
	Indexes    []statement.Index
	Options    []*CreateTableOptionStatement
}

type CreateTableOptionStatement struct {
	Key   string
	Value string
}

type MaybeString struct {
	Valid bool
	Value string
}

const (
	coloptSize = 1 << iota
	coloptDecimalSize
	coloptDecimalOptionalSize
	coloptUnsigned
	coloptZerofill
	coloptBinary
	coloptCharacterSet
	coloptCollate
	coloptNull
	coloptDefault
	coloptAutoIncrement
	coloptKey
	coloptComment
)

const (
	coloptFlagNone            = 0
	coloptFlagDigit           = coloptSize | coloptUnsigned | coloptZerofill
	coloptFlagDecimal         = coloptDecimalSize | coloptUnsigned | coloptZerofill
	coloptFlagDecimalOptional = coloptDecimalOptionalSize | coloptUnsigned | coloptZerofill
	coloptFlagTime            = coloptSize
	coloptFlagChar            = coloptSize | coloptBinary | coloptCharacterSet | coloptCollate
	coloptFlagBinary          = coloptSize
)

