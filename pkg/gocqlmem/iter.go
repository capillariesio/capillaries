package gocqlmem

import (
	"fmt"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
)

type gocqlmemIter struct {
	err error
	pos int
	//meta    resultMetadata
	//numRows int
	//next    *nextIter
	//host    *HostInfo

	//framer *framer
	//closed int32

	keyspace             string
	table                string
	retrievedValues      [][]any
	retrievedColumnInfos []gocql.ColumnInfo
	pagingState          []byte
}

func NewGocqlmemIterWithError(err error) *gocqlmemIter {
	return &gocqlmemIter{err: err}
}
func NewGocqlmemIterWithKeyspace(ks string) *gocqlmemIter {
	return &gocqlmemIter{keyspace: ks}
}

func NewGocqlmemIterWithData(ks string, table string, infos []gocql.ColumnInfo, values [][]any) *gocqlmemIter {
	return &gocqlmemIter{keyspace: ks, table: table, retrievedColumnInfos: infos, retrievedValues: values}
}

func NewGocqlmemIterWithDataAndPagingState(ks string, table string, infos []gocql.ColumnInfo, values [][]any, pagingState []byte) *gocqlmemIter {
	return &gocqlmemIter{keyspace: ks, table: table, retrievedColumnInfos: infos, retrievedValues: values, pagingState: pagingState}
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
	return &iterScanner{Iter: iter, Cols: make([]interface{}, len(iter.retrievedColumnInfos))}
}

func (iter *gocqlmemIter) Scan(dest ...interface{}) bool {
	if iter.err != nil || iter.pos >= len(iter.retrievedValues) {
		return false
	}

	if len(dest) != len(iter.retrievedColumnInfos) {
		iter.SetErr(fmt.Errorf("gocqlmem: not enough columns to scan into: have %d want %d", len(dest), len(iter.retrievedColumnInfos)))
		return false
	}

	for i := range len(iter.retrievedColumnInfos) {
		dest[i] = iter.retrievedValues[iter.pos][i]
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
		Values:  make([]interface{}, len(iter.retrievedColumnInfos)),
	}

	for i := range len(iter.retrievedColumnInfos) {
		if iter.retrievedColumnInfos[i].TypeInfo == nil {
			// We could not guess the type this expression, so do not initialize this value
			continue
		}
		rowData.Values[i] = iter.retrievedColumnInfos[i].TypeInfo.Zero()
	}

	return rowData, nil
}

func (iter *gocqlmemIter) SliceMap() ([]map[string]interface{}, error) {
	if iter.err != nil {
		return nil, iter.err
	}

	// Not checking for the error because we just did
	rowData, _ := iter.RowData()
	dataToReturn := make([]map[string]interface{}, 0)
	for iter.Scan(rowData.Values...) {
		m := make(map[string]interface{}, len(rowData.Columns))
		for i, column := range rowData.Columns {
			m[column] = rowData.Values[i]
		}
		dataToReturn = append(dataToReturn, m)
	}
	if iter.err != nil {
		return nil, iter.err
	}
	return dataToReturn, nil
}

func (iter *gocqlmemIter) MapScan(dest map[string]interface{}) bool {
	if iter.err != nil {
		return false
	}

	if dest != nil {
		clear(dest)
	}

	rowDataValues := make([]any, len(iter.retrievedColumnInfos))
	if iter.Scan(rowDataValues...) {
		if dest != nil {
			for i, columnInfo := range iter.retrievedColumnInfos {
				dest[columnInfo.Name] = rowDataValues[i]
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
