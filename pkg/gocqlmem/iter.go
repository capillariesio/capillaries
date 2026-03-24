package gocqlmem

import (
	"fmt"
	"time"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
	"gopkg.in/inf.v0"
)

type gocqlmemIter struct {
	err error
	pos int
	// meta    resultMetadata
	// numRows int
	// next    *nextIter
	// host    *HostInfo

	// framer *framer
	// closed int32

	keyspace             string
	table                string
	retrievedValues      [][]any
	retrievedColumnInfos []gocql.ColumnInfo
	pagingState          []byte
}

/*
func (iter *gocqlmemIter) guessTypeInfosFromData() error {
	for colIdx := range len(iter.retrievedColumnInfos) {
		if iter.retrievedColumnInfos[colIdx].TypeInfo != nil {
			// Type was provided from column def or guessed by guessInternalValueType
			continue
		}
		// retrieved values for that column was nil, try guessing the client type here by walking through all values
		for rowIdx := range len(iter.retrievedValues) {
			if iter.retrievedValues[rowIdx][colIdx] == nil {
				continue
			}
			return fmt.Errorf("bla")
			typ, err := guessClientValueType(iter.retrievedValues[rowIdx][colIdx])
			if err != nil {
				return err
			}
			iter.retrievedColumnInfos[colIdx].TypeInfo = &scalarType{typ: typ}
			break
		}
	}
	return nil // WARNING: if all retrieved values for a column X are nil, correspondent TypeInfo will be nil
}
*/

func newGocqlmemIterWithError(err error) *gocqlmemIter {
	return &gocqlmemIter{err: err}
}
func newGocqlmemIterWithKeyspace(ks string) *gocqlmemIter {
	return &gocqlmemIter{keyspace: ks}
}

func newGocqlmemIterWithData(ks string, table string, infos []gocql.ColumnInfo, values [][]any) *gocqlmemIter {
	// WARNING: if all retrieved values for a column X are nil, correspondent TypeInfo will be nil
	iter := gocqlmemIter{keyspace: ks, table: table, retrievedColumnInfos: infos, retrievedValues: values}
	return &iter
}

func newGocqlmemIterWithDataAndPagingState(ks string, table string, infos []gocql.ColumnInfo, values [][]any, pagingState []byte) *gocqlmemIter {
	// WARNING: if all retrieved values for a column X are nil, correspondent TypeInfo will be nil
	iter := gocqlmemIter{keyspace: ks, table: table, retrievedColumnInfos: infos, retrievedValues: values, pagingState: pagingState}
	return &iter
}

// Iter interface

func (iter *gocqlmemIter) Host() *gocql.HostInfo {
	// TODO: implement
	return &gocql.HostInfo{}
}

func (iter *gocqlmemIter) Columns() []gocql.ColumnInfo {
	return iter.retrievedColumnInfos
}

func (iter *gocqlmemIter) Attempts() int {
	// TODO: implement
	return 1
}

func (iter *gocqlmemIter) Latency() int64 {
	// TODO: implement
	return 0
}

func (iter *gocqlmemIter) Keyspace() string {
	return iter.keyspace
}

func (iter *gocqlmemIter) Table() string {
	return iter.table
}

func (iter *gocqlmemIter) Scanner() gocql.Scanner {
	if iter == nil {
		return nil
	}
	return &iterScanner{Iter: iter, Cols: make([]any, len(iter.retrievedColumnInfos))}
}

func (iter *gocqlmemIter) Scan(dest ...any) bool {
	if iter.err != nil || iter.pos >= len(iter.retrievedValues) {
		return false
	}

	if len(dest) != len(iter.retrievedColumnInfos) {
		iter.SetErr(fmt.Errorf("gocqlmem: not enough columns to scan into: have %d want %d", len(dest), len(iter.retrievedColumnInfos)))
		return false
	}

	for i := range len(iter.retrievedColumnInfos) {
		if dest[i] != nil {
			if iter.retrievedValues[iter.pos][i] == nil {
				dest[i] = nil
			} else {
				if err := clientTypedValueToProvidedPtr(iter.retrievedValues[iter.pos][i], dest[i]); err != nil {
					iter.SetErr(fmt.Errorf("cannot scan column %d: %s", i, err.Error()))
					return false
				}
			}
		}
	}

	iter.pos++
	return true

	// if iter.err != nil {
	// 	return false
	// }

	// if iter.pos >= iter.numRows {
	// 	if iter.next != nil {
	// 		*iter = *iter.next.fetch()
	// 		return iter.Scan(dest...)
	// 	}
	// 	return false
	// }

	// if iter.next != nil && iter.pos >= iter.next.pos {
	// 	iter.next.fetchAsync()
	// }

	// // currently only support scanning into an expand tuple, such that its the same
	// // as scanning in more values from a single column
	// if len(dest) != iter.meta.actualColCount {
	// 	iter.err = fmt.Errorf("gocql: not enough columns to scan into: have %d want %d", len(dest), iter.meta.actualColCount)
	// 	return false
	// }

	// // i is the current position in dest, could posible replace it and just use
	// // slices of dest
	// i := 0
	// for _, col := range iter.meta.columns {
	// 	colBytes, err := iter.readColumn()
	// 	if err != nil {
	// 		iter.err = err
	// 		return false
	// 	}

	// 	n, err := scanColumn(colBytes, col, dest[i:])
	// 	if err != nil {
	// 		iter.err = err
	// 		return false
	// 	}
	// 	i += n
	// }

	// iter.pos++
	// return true
}

func (iter *gocqlmemIter) GetCustomPayload() map[string][]byte {
	// TODO: implement
	return map[string][]byte{}
}
func (iter *gocqlmemIter) Warnings() []string {
	// TODO: implement
	return []string{}
}

func (iter *gocqlmemIter) Close() error {
	return iter.err
}

func (iter *gocqlmemIter) WillSwitchPage() bool {
	// TODO: implement
	return false
}

func (iter *gocqlmemIter) PageState() []byte {
	return iter.pagingState
}

func (iter *gocqlmemIter) NumRows() int {
	return len(iter.retrievedValues)
}

// Do not ask me why gocql exposes this
func (iter *gocqlmemIter) RowData() (gocql.RowData, error) {
	if iter.err != nil {
		return gocql.RowData{}, iter.err
	}

	rowData := gocql.RowData{
		Columns: columnInfosToColumnNames(iter.retrievedColumnInfos),
		Values:  make([]any, len(iter.retrievedColumnInfos)),
	}

	for colIdx := range len(iter.retrievedColumnInfos) {
		if iter.retrievedColumnInfos[colIdx].TypeInfo == nil {
			// Is the caller prepared for for a nil ptr?
			rowData.Values[colIdx] = nil
		} else {
			rowData.Values[colIdx] = iter.retrievedColumnInfos[colIdx].TypeInfo.Zero()
		}
	}

	return rowData, nil
}

func (iter *gocqlmemIter) SliceMap() ([]map[string]any, error) {
	if iter.err != nil {
		return nil, iter.err
	}

	totalRows := len(iter.retrievedValues) - iter.pos
	result := make([]map[string]any, totalRows)
	for rowIdx := range totalRows {
		result[rowIdx] = map[string]any{}
		for colIdx, colInfo := range iter.retrievedColumnInfos {
			result[rowIdx][colInfo.Name] = iter.retrievedValues[rowIdx][colIdx]
		}
		iter.pos++
	}

	return result, nil

	// // Not checking for the error because we just did

	// TODO: prepare rowdata values before each scan. If typeinfo not available, use iter.retrievedValues and iter.pos to obtain the gocql.Type
	// init  rowData.Columns only once

	// rowData, _ := iter.RowData()
	// dataToReturn := make([]map[string]any, 0)
	// for iter.Scan(rowData.Values...) {
	// 	m := make(map[string]any, len(rowData.Columns))
	// 	for i, column := range rowData.Columns {
	// 		switch typedVal := rowData.Values[i].(type) {
	// 		case *int:
	// 			m[column] = *typedVal
	// 		case *int8:
	// 			m[column] = *typedVal
	// 		case *int16:
	// 			m[column] = *typedVal
	// 		case *int32:
	// 			m[column] = *typedVal
	// 		case *int64:
	// 			m[column] = *typedVal
	// 		case *float32:
	// 			m[column] = *typedVal
	// 		case *float64:
	// 			m[column] = *typedVal
	// 		case *string:
	// 			m[column] = *typedVal
	// 		case *bool:
	// 			m[column] = *typedVal
	// 		case *gocql.UUID:
	// 			m[column] = *typedVal
	// 		case *inf.Dec:
	// 			m[column] = *typedVal
	// 		case *time.Time:
	// 			m[column] = *typedVal
	// 		}
	// 	}
	// 	dataToReturn = append(dataToReturn, m)
	// }
	// if iter.err != nil {
	// 	return nil, iter.err
	// }
	// return dataToReturn, nil
}

func (iter *gocqlmemIter) MapScan(dest map[string]any) bool {
	if iter.err != nil {
		return false
	}

	if dest != nil {
		clear(dest)
	}

	rowData, _ := iter.RowData()
	if iter.Scan(rowData.Values...) {
		if dest != nil {
			for i, columnInfo := range iter.retrievedColumnInfos {
				switch typedVal := rowData.Values[i].(type) {
				case *int:
					dest[columnInfo.Name] = *typedVal
				case *int8:
					dest[columnInfo.Name] = *typedVal
				case *int16:
					dest[columnInfo.Name] = *typedVal
				case *int32:
					dest[columnInfo.Name] = *typedVal
				case *int64:
					dest[columnInfo.Name] = *typedVal
				case *float32:
					dest[columnInfo.Name] = *typedVal
				case *float64:
					dest[columnInfo.Name] = *typedVal
				case *string:
					dest[columnInfo.Name] = *typedVal
				case *bool:
					dest[columnInfo.Name] = *typedVal
				case *gocql.UUID:
					dest[columnInfo.Name] = *typedVal
				case *inf.Dec:
					dest[columnInfo.Name] = *typedVal
				case *time.Time:
					dest[columnInfo.Name] = *typedVal
				default:
					// TODO: handle other types
				}
			}
		}
		return true
	}
	return false

	// // Not checking for the error because we just did
	// rowData, _ := iter.RowData()

	// for i, col := range rowData.Columns {
	// 	if dest, ok := m[col]; ok {
	// 		rowData.Values[i] = dest
	// 	}
	// }

	// if iter.Scan(rowData.Values...) {
	// 	rowData.rowMap(m)
	// 	return true
	// }
	// return false
}

func (iter *gocqlmemIter) Err() error {
	return iter.err
}

func (iter *gocqlmemIter) SetErr(err error) {
	iter.err = err
}
