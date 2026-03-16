package gocqlmem

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func sliceMapRowsToString(rows []map[string]any) string {
	if rows == nil {
		return ""
	}
	sb := strings.Builder{}
	sb.WriteString("[")
	for i, r := range rows {
		if i > 0 {
			sb.WriteString("]")
		}
		sb.WriteString(fmt.Sprintf("%v", r))
	}
	sb.WriteString("]")
	return sb.String()
}

func assertIterSliceMap(t *testing.T, expectedRows string, expectedErr string, s Session, q string) {
	rows, err := s.Query(q).Iter().SliceMap()
	if expectedErr == "" {
		assert.Nil(t, err)
	} else {
		assert.Equal(t, expectedErr, err.Error())
	}
	assert.Equal(t, expectedRows, sliceMapRowsToString(rows))
}

func assertIterScan(t *testing.T, expectedRows string, s Session, q string) {
	iter := s.Query(q).Iter()
	assert.Nil(t, iter.Err())
	sb := strings.Builder{}
	rd, _ := iter.RowData()
	sb.WriteString("[")
	for iter.Scan(rd.Values...) {
		sb.WriteString("[")
		for valIdx, valPtr := range rd.Values {
			if valIdx > 0 {
				sb.WriteString(",")
			}
			sb.WriteString(clientValuePtrToString(valPtr))
		}
		sb.WriteString("]")
	}
	sb.WriteString("]")
	assert.Equal(t, expectedRows, sb.String())
}

func assertIterMapScan(t *testing.T, expectedRows string, s Session, q string) {
	iter := s.Query(q).Iter()
	sb := strings.Builder{}
	row := map[string]interface{}{}
	for _, colInfo := range iter.Columns() {
		row[colInfo.Name] = nil
	}
	sb.WriteString("[")
	for iter.MapScan(row) {
		sb.WriteString(fmt.Sprintf("[%v]", row))
	}
	sb.WriteString("]")
	assert.Equal(t, expectedRows, sb.String())
}

func assertScanner(t *testing.T, expectedRows string, s Session, q string) {
	iter := s.Query(q).Iter()
	row := make([]int64, len(iter.Columns()))
	sb := strings.Builder{}
	scanner := iter.Scanner()
	sb.WriteString("[")
	for scanner.Next() {
		scanner.Scan(row)
		sb.WriteString("[")
		for valIdx, val := range row {
			if valIdx > 0 {
				sb.WriteString(",")
			}
			sb.WriteString(fmt.Sprintf("%v", val))
		}
		sb.WriteString("]")
	}
	sb.WriteString("]")
	assert.Equal(t, expectedRows, sb.String())
}

func assertUpserMapScanCas(t *testing.T, isApplyExpected bool, s Session, q string, existingRowMap map[string]interface{}) {
	isApplied, err := s.Query(q).MapScanCAS(existingRowMap)
	assert.Nil(t, err)
	assert.Equal(t, isApplyExpected, isApplied)
}
