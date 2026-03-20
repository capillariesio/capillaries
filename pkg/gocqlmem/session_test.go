package gocqlmem

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	gocql "github.com/apache/cassandra-gocql-driver/v2"
	"github.com/stretchr/testify/assert"
	"gopkg.in/inf.v0"
)

func clientValuePtrToString(val any) string {
	switch typedVal := val.(type) {
	case *int:
		return strconv.FormatInt(int64(*typedVal), 10)
	case *int8:
		return strconv.FormatInt(int64(*typedVal), 10)
	case *int16:
		return strconv.FormatInt(int64(*typedVal), 10)
	case *int32:
		return strconv.FormatInt(int64(*typedVal), 10)
	case *int64:
		return strconv.FormatInt(*typedVal, 10)
	case *float32:
		return strconv.FormatFloat(float64(*typedVal), 'f', -1, 32)
	case *float64:
		return strconv.FormatFloat(*typedVal, 'f', -1, 64)
	case *bool:
		return strconv.FormatBool(*typedVal)
	case *string:
		return *typedVal
	case *inf.Dec:
		return typedVal.String()
	case *gocql.UUID:
		return fmt.Sprintf("%x-%x-%x-%x-%x", typedVal[0:4], typedVal[4:6], typedVal[6:8], typedVal[8:10], typedVal[10:16])
	case *time.Time:
		return typedVal.Format(time.RFC3339)
	case *[]byte:
		return fmt.Sprintf("%x", typedVal)
	default:
		return fmt.Sprintf("%v", typedVal)
	}
}

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
