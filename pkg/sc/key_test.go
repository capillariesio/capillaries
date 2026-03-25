package sc

import (
	"maps"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/capillariesio/capillaries/pkg/evalcapi"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func assertKeyErrorPrefix(t *testing.T, expectedErrorPrefix string, actualError string) {
	if !strings.HasPrefix(actualError, expectedErrorPrefix) {
		t.Errorf("\nExpected error prefix:\n%s\nGot error:\n%s", expectedErrorPrefix, actualError)
	}
}

func assertKeyCompare(
	t *testing.T,
	row1 map[string]any,
	moreLess string,
	row2 map[string]any,
	idxDef IdxDef) {

	if moreLess != "<" && moreLess != ">" && moreLess != "==" {
		t.Errorf("Invalid moreLess value: %v", moreLess)
		return
	}

	key1, err1 := BuildKey(row1, &idxDef)
	if err1 != nil {
		t.Errorf("cannot build key1 %s\n", err1)
	}

	key2, err2 := BuildKey(row2, &idxDef)
	if err2 != nil {
		t.Errorf("cannot build key2 %s\n", err2)
	}

	if moreLess == "<" && (key1 >= key2) || moreLess == ">" && (key1 <= key2) || moreLess == "==" && (key1 != key2) {
		t.Errorf("\nExpected:\n%s\n%s\n%s\nrow1: %v\nrow2: %v\n", key1, moreLess, key2, row1, row2)
	}
}

func TestBad(t *testing.T) {

	idxDef := IdxDef{Uniqueness: "UNIQUE", Components: []IdxComponentDef{{FieldName: "fld", SortOrder: IdxSortAsc, FieldType: evalcapi.FieldTypeInt}}}
	row1 := map[string]any{"fld": false}

	_, err := BuildKey(row1, &idxDef)
	assert.Equal(t, "cannot convert value false to type int", err.Error())

	idxDef.Components[0].FieldType = evalcapi.FieldTypeFloat
	_, err = BuildKey(row1, &idxDef)
	assert.Equal(t, "cannot convert value false to type float", err.Error())

	idxDef.Components[0].FieldType = evalcapi.FieldTypeDecimal2
	_, err = BuildKey(row1, &idxDef)
	assert.Equal(t, "cannot convert value false to type decimal2", err.Error())

	idxDef.Components[0].FieldType = evalcapi.FieldTypeDateTime
	_, err = BuildKey(row1, &idxDef)
	assert.Equal(t, "cannot convert value false to type datetime", err.Error())

	idxDef.Components[0].FieldType = evalcapi.FieldTypeString
	_, err = BuildKey(row1, &idxDef)
	assert.Equal(t, "cannot convert value false to type string", err.Error())

	idxDef.Components[0].FieldType = evalcapi.FieldTypeBool
	row1["fld"] = int64(2)
	_, err = BuildKey(row1, &idxDef)
	assert.Equal(t, "cannot convert value 2 to type bool", err.Error())

	idxDef.Components[0].FieldType = evalcapi.FieldTypeUnknown
	_, err = BuildKey(row1, &idxDef)
	assert.Equal(t, "cannot build key, unsupported field data type unknown", err.Error())
}

func TestCombined(t *testing.T) {

	idxDef := IdxDef{
		Uniqueness: "UNIQUE",
		Components: []IdxComponentDef{
			{
				FieldName:       "field_int",
				CaseSensitivity: IdxCaseSensitivityUnknown,
				FieldType:       evalcapi.FieldTypeInt,
			},
			{
				FieldName:       "field_string",
				CaseSensitivity: IdxIgnoreCase,
				FieldType:       evalcapi.FieldTypeString,
				StringLen:       64,
			},
			{
				FieldName:       "field_float",
				CaseSensitivity: IdxCaseSensitivityUnknown,
				FieldType:       evalcapi.FieldTypeFloat,
			},
			{
				FieldName:       "field_bool",
				CaseSensitivity: IdxCaseSensitivityUnknown,
				FieldType:       evalcapi.FieldTypeBool,
			},
		},
	}

	baseRow1 := map[string]any{
		"field_int":    int64(1),
		"field_string": "abc",
		"field_float":  -2.3,
		"field_bool":   false,
	}
	baseRow2 := map[string]any{
		"field_int":    int64(1),
		"field_string": "Abc",
		"field_float":  1.3,
		"field_bool":   true,
	}

	var row1, row2 map[string]any

	// -2.3 < 1.3
	row1 = maps.Clone(baseRow1)
	row2 = maps.Clone(baseRow2)
	assertKeyCompare(t, row1, "<", row2, idxDef)

	// -2.3 < -2.0
	row1 = maps.Clone(baseRow1)
	row2 = maps.Clone(baseRow2)
	row2["field_float"] = -2.0
	assertKeyCompare(t, row1, "<", row2, idxDef)

	// abc > Abc
	idxDef.Components[1].CaseSensitivity = IdxCaseSensitive
	assertKeyCompare(t, row1, ">", row2, idxDef)

	// F < T
	row1 = maps.Clone(baseRow1)
	row2 = maps.Clone(baseRow2)
	row2["field_string"] = row1["field_string"]
	row2["field_float"] = row1["field_float"]
	assertKeyCompare(t, row1, "<", row2, idxDef)

	// F == F
	row1 = maps.Clone(baseRow1)
	row2 = maps.Clone(baseRow2)
	row2["field_bool"] = row1["field_bool"]
	row2["field_string"] = row1["field_string"]
	row2["field_float"] = row1["field_float"]
	assertKeyCompare(t, row1, "==", row2, idxDef)

	// -3786697372163639434 < -416149536780825218 (number of digits)
	row1 = maps.Clone(baseRow1)
	row2 = maps.Clone(baseRow2)
	row1["field_int"] = int64(-3786697372163639434)
	row2["field_int"] = int64(-416149536780825218)
	assertKeyCompare(t, row1, "<", row2, idxDef)

	// 123 > 99 (number of digits)
	row1 = maps.Clone(baseRow1)
	row2 = maps.Clone(baseRow2)
	row1["field_int"] = int64(123)
	row2["field_int"] = int64(99)
	assertKeyCompare(t, row1, ">", row2, idxDef)

	// No such field in the table row
	row1 = maps.Clone(baseRow1)
	delete(row1, "field_float")
	_, err2 := BuildKey(row1, &idxDef)
	assertKeyErrorPrefix(t, "cannot find value for field field_float in", err2.Error())
}

func TestTime(t *testing.T) {

	idxDef := IdxDef{
		Uniqueness: "UNIQUE",
		Components: []IdxComponentDef{{FieldName: "fld", FieldType: evalcapi.FieldTypeDateTime}},
	}

	idxDef.Components[0].SortOrder = IdxSortAsc

	row1 := map[string]any{"fld": time.Date(1, time.January, 1, 2, 2, 2, 3000, time.UTC)}
	row2 := map[string]any{"fld": time.Date(1, time.January, 1, 2, 2, 2, 4000, time.UTC)}
	assertKeyCompare(t, row1, "<", row2, idxDef)

	row1 = map[string]any{"fld": time.Date(850000, time.January, 1, 2, 2, 2, 3000, time.UTC)}
	row2 = map[string]any{"fld": time.Date(850000, time.January, 1, 2, 2, 2, 4000, time.UTC)}
	assertKeyCompare(t, row1, "<", row2, idxDef)

	idxDef.Components[0].SortOrder = IdxSortDesc

	assertKeyCompare(t, row1, ">", row2, idxDef)
}

func TestBool(t *testing.T) {

	idxDef := IdxDef{
		Uniqueness: "UNIQUE",
		Components: []IdxComponentDef{{FieldName: "fld", FieldType: evalcapi.FieldTypeBool}},
	}

	row1 := map[string]any{"fld": false}
	row2 := map[string]any{"fld": true}

	idxDef.Components[0].SortOrder = IdxSortAsc
	assertKeyCompare(t, row1, "<", row2, idxDef)

	idxDef.Components[0].SortOrder = IdxSortDesc
	assertKeyCompare(t, row1, ">", row2, idxDef)
}

func TestInt(t *testing.T) {

	idxDef := IdxDef{
		Uniqueness: "UNIQUE",
		Components: []IdxComponentDef{{FieldName: "fld", FieldType: evalcapi.FieldTypeInt}},
	}

	idxDef.Components[0].SortOrder = IdxSortAsc

	row1 := map[string]any{"fld": int64(1000)}
	row2 := map[string]any{"fld": int64(2000)}
	assertKeyCompare(t, row1, "<", row2, idxDef)

	row1 = map[string]any{"fld": int64(-1000)}
	row2 = map[string]any{"fld": int64(-2000)}
	assertKeyCompare(t, row1, ">", row2, idxDef)

	row1 = map[string]any{"fld": int64(-1000)}
	row2 = map[string]any{"fld": int64(50)}
	assertKeyCompare(t, row1, "<", row2, idxDef)

	idxDef.Components[0].SortOrder = IdxSortDesc

	row1 = map[string]any{"fld": int64(-1000)}
	row2 = map[string]any{"fld": int64(50)}
	assertKeyCompare(t, row1, ">", row2, idxDef)

}
func TestFloat(t *testing.T) {

	idxDef := IdxDef{
		Uniqueness: "UNIQUE",
		Components: []IdxComponentDef{{FieldName: "fld", FieldType: evalcapi.FieldTypeFloat}},
	}

	idxDef.Components[0].SortOrder = IdxSortAsc

	row1 := map[string]any{"fld": 1.1}
	row2 := map[string]any{"fld": 1.2}
	assertKeyCompare(t, row1, "<", row2, idxDef)

	row1 = map[string]any{"fld": math.Pow10(32)}
	row2 = map[string]any{"fld": math.Pow10(32) / 2}
	assertKeyCompare(t, row1, ">", row2, idxDef)

	row1 = map[string]any{"fld": -math.Pow10(32)}
	row2 = map[string]any{"fld": -math.Pow10(32) / 2}
	assertKeyCompare(t, row1, "<", row2, idxDef)

	row1 = map[string]any{"fld": math.Pow10(-32)}
	row2 = map[string]any{"fld": math.Pow10(-32) * 2}
	assertKeyCompare(t, row1, "<", row2, idxDef)

	row1 = map[string]any{"fld": -math.Pow10(-32)}
	row2 = map[string]any{"fld": -math.Pow10(-32) * 2}
	assertKeyCompare(t, row1, ">", row2, idxDef)

	row1 = map[string]any{"fld": -1.2}
	row2 = map[string]any{"fld": 0.005}
	assertKeyCompare(t, row1, "<", row2, idxDef)

	idxDef.Components[0].SortOrder = IdxSortDesc

	row1 = map[string]any{"fld": 1.1}
	row2 = map[string]any{"fld": 1.2}
	assertKeyCompare(t, row1, ">", row2, idxDef)
}

func TestString(t *testing.T) {

	// Use MinStringComponentLen = 16
	idxDef := IdxDef{
		Uniqueness: "UNIQUE",
		Components: []IdxComponentDef{{FieldName: "fld", CaseSensitivity: IdxIgnoreCase, FieldType: evalcapi.FieldTypeString, StringLen: 16}},
	}

	idxDef.Components[0].SortOrder = IdxSortAsc

	// Different length
	row1 := map[string]any{"fld": "aaa"}
	row2 := map[string]any{"fld": "bb"}
	assertKeyCompare(t, row1, "<", row2, idxDef)

	// Plain
	row1 = map[string]any{"fld": "aaa"}
	row2 = map[string]any{"fld": "bbb"}
	assertKeyCompare(t, row1, "<", row2, idxDef)

	// Ignore case
	row1 = map[string]any{"fld": "aaa"}
	row2 = map[string]any{"fld": "Abb"}
	assertKeyCompare(t, row1, "<", row2, idxDef)

	// Beyond StringLen
	row1 = map[string]any{"fld": "1234567890123456A"}
	row2 = map[string]any{"fld": "1234567890123456B"}
	assertKeyCompare(t, row1, "==", row2, idxDef)

	// Within StringLen
	row1 = map[string]any{"fld": "123456789012345A"}
	row2 = map[string]any{"fld": "123456789012345B"}
	assertKeyCompare(t, row1, "<", row2, idxDef)

	idxDef.Components[0].SortOrder = IdxSortDesc

	// Reverse order
	row1 = map[string]any{"fld": "aaa"}
	row2 = map[string]any{"fld": "bbb"}
	assertKeyCompare(t, row1, ">", row2, idxDef)
}

func TestDecimal(t *testing.T) {

	idxDef := IdxDef{
		Uniqueness: "UNIQUE",
		Components: []IdxComponentDef{{FieldName: "fld", FieldType: evalcapi.FieldTypeDecimal2}}}

	idxDef.Components[0].SortOrder = IdxSortAsc

	row1 := map[string]any{"fld": decimal.NewFromFloat32(0.23456)}
	row2 := map[string]any{"fld": decimal.NewFromFloat32(985.4)}
	assertKeyCompare(t, row1, "<", row2, idxDef)

	row1 = map[string]any{"fld": decimal.NewFromFloat32(0.23456)}
	row2 = map[string]any{"fld": decimal.NewFromFloat32(-985.4)}
	assertKeyCompare(t, row1, ">", row2, idxDef)

	row1 = map[string]any{"fld": decimal.NewFromFloat32(0.002)}
	row2 = map[string]any{"fld": decimal.NewFromFloat32(0.01)}
	assertKeyCompare(t, row1, "<", row2, idxDef)

	row1 = map[string]any{"fld": decimal.NewFromFloat32(-2000)}
	row2 = map[string]any{"fld": decimal.NewFromFloat32(-1000)}
	assertKeyCompare(t, row1, "<", row2, idxDef)

	idxDef.Components[0].SortOrder = IdxSortDesc

	row1 = map[string]any{"fld": decimal.NewFromFloat32(0.23456)}
	row2 = map[string]any{"fld": decimal.NewFromFloat32(985.4)}
	assertKeyCompare(t, row1, ">", row2, idxDef)

	row1 = map[string]any{"fld": decimal.NewFromFloat32(0.23456)}
	row2 = map[string]any{"fld": decimal.NewFromFloat32(-985.4)}
	assertKeyCompare(t, row1, "<", row2, idxDef)

	row1 = map[string]any{"fld": decimal.NewFromFloat32(0.002)}
	row2 = map[string]any{"fld": decimal.NewFromFloat32(0.01)}
	assertKeyCompare(t, row1, ">", row2, idxDef)

	row1 = map[string]any{"fld": decimal.NewFromFloat32(-2000)}
	row2 = map[string]any{"fld": decimal.NewFromFloat32(-1000)}
	assertKeyCompare(t, row1, ">", row2, idxDef)
}

func TestGetNUmericValueSign(t *testing.T) {
	_, _, err := getNumericValueSign(nil, evalcapi.FieldTypeUnknown)
	assert.Contains(t, err.Error(), "cannot convert value <nil> to type unknown")
}
