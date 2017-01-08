package schemalex

import "io"

type Stmt interface {
	WriteTo(io.Writer) (int64, error)
}

type Statements []Stmt

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
