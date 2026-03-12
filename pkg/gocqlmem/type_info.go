package gocqlmem

import (
	gocql "github.com/apache/cassandra-gocql-driver/v2"
	"gopkg.in/inf.v0"
)

type CqlDataType string

const (
	DataTypeAscii     CqlDataType = "ascii"
	DataTypeBigint    CqlDataType = "bigint"
	DataTypeBlob      CqlDataType = "blob"
	DataTypeBoolean   CqlDataType = "boolean"
	DataTypeCounter   CqlDataType = "counter"
	DataTypeDate      CqlDataType = "date"
	DataTypeDecimal   CqlDataType = "decimal"
	DataTypeDouble    CqlDataType = "double"
	DataTypeDuration  CqlDataType = "duration"
	DataTypeFloat     CqlDataType = "float"
	DataTypeInet      CqlDataType = "inet"
	DataTypeInt       CqlDataType = "int"
	DataTypeSmallint  CqlDataType = "smallint"
	DataTypeText      CqlDataType = "text"
	DataTypeTime      CqlDataType = "time"
	DataTypeTimestamp CqlDataType = "timestamp"
	DataTypeTimeuuid  CqlDataType = "timeuuid"
	DataTypeTinyint   CqlDataType = "tinyint"
	DataTypeUuid      CqlDataType = "uuid"
	DataTypeVarchar   CqlDataType = "varchar"
	DataTypeVarint    CqlDataType = "varint"
	DataTypeUnknown   CqlDataType = "unknown"
)

// gocql has a group of types like varcharLikeTypeInfo etc, but we are ok with just one for now
type scalarType struct {
	typ gocql.Type
}

func newScalarType(typ gocql.Type) *scalarType {
	return &scalarType{typ: typ}
}

// Implement gocql TypeInfo interface
func (t *scalarType) Type() gocql.Type {
	return t.typ
}

func (t *scalarType) Zero() interface{} {
	switch t.typ {
	case gocql.TypeInt, gocql.TypeBigInt, gocql.TypeSmallInt, gocql.TypeTinyInt:
		return int64(0)
	case gocql.TypeFloat:
		return float64(0)
	case gocql.TypeText, gocql.TypeVarchar, gocql.TypeAscii:
		return ""
	case gocql.TypeBoolean:
		return false
	case gocql.TypeDecimal:
		return *inf.NewDec(0, 0)
	case gocql.TypeBlob:
		return []byte(nil)
	default:
		// TODO: raise an alarm
		return nil
	}
}

func (t *scalarType) Marshal(value interface{}) ([]byte, error) {
	// Not implemented, do we need it in our project?
	return nil, nil
}

func (t *scalarType) Unmarshal(data []byte, value interface{}) error {
	// Not implemented, do we need it in our project?
	return nil
}
