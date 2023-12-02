package sc

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/capillariesio/capillaries/pkg/eval"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func assertErrorPrefix(t *testing.T, expectedErrorPrefix string, actualError string) {
	if !strings.HasPrefix(actualError, expectedErrorPrefix) {
		t.Errorf("\nExpected error prefix:\n%s\nGot error:\n%s", expectedErrorPrefix, actualError)
	}
}

func testReader(fileReaderJson string, srcLine []string) (eval.VarValuesMap, error) {
	fileReader := FileReaderDef{}
	if err := fileReader.Deserialize([]byte(fileReaderJson)); err != nil {
		return nil, err
	}

	if fileReader.Csv.ColumnIndexingMode == FileColumnIndexingName {
		srcHdrLine := []string{"order_id", "customer_id", "order_status", "order_purchase_timestamp"}
		if err := fileReader.ResolveCsvColumnIndexesFromNames(srcHdrLine); err != nil {
			return nil, err
		}
	}

	colRecord := eval.VarValuesMap{}
	if err := fileReader.ReadCsvLineToValuesMap(&srcLine, colRecord); err != nil {
		return nil, err
	}

	return colRecord, nil
}

func TestFieldRefs(t *testing.T) {
	conf := `
	{
		"urls": [""],
		"csv":{
			"hdr_line_idx": 0,
			"first_data_line_idx": 1
		},
		"columns":  {
			"col_order_id": {
				"csv":{
					"col_idx": 0,
					"col_hdr": null
				},
				"col_type": "string"
			},
			"col_order_status": {
				"csv":{
					"col_idx": 2,
					"col_hdr": null
				},
				"col_type": "string"
			},
			"col_order_purchase_timestamp": {
				"csv":{
					"col_idx": 3,
					"col_hdr": null,
					"col_format": "2006-01-02 15:04:05"
				},
				"col_type": "datetime"
			}
		}
	}`
	reader := FileReaderDef{}
	assert.Nil(t, reader.Deserialize([]byte(conf)))

	fieldRefs := reader.getFieldRefs()
	var fr *FieldRef
	fr, _ = fieldRefs.FindByFieldName("col_order_id")
	assert.Equal(t, ReaderAlias, fr.TableName)
	assert.Equal(t, FieldTypeString, fr.FieldType)
	fr, _ = fieldRefs.FindByFieldName("col_order_status")
	assert.Equal(t, ReaderAlias, fr.TableName)
	assert.Equal(t, FieldTypeString, fr.FieldType)
	fr, _ = fieldRefs.FindByFieldName("col_order_purchase_timestamp")
	assert.Equal(t, ReaderAlias, fr.TableName)
	assert.Equal(t, FieldTypeDateTime, fr.FieldType)
}

func TestColumnIndexing(t *testing.T) {
	srcLine := []string{"order_id_1", "customer_id_1", "delivered", "2017-10-02 10:56:33"}

	// Good by idx
	colRecord, err := testReader(`
		{
			"urls": [""],
			"csv":{
				"hdr_line_idx": 0,
				"first_data_line_idx": 1
			},
			"columns":  {
				"col_order_id": {
					"csv":{
						"col_idx": 0,
						"col_hdr": null
					},
					"col_type": "string"
				},
				"col_order_status": {
					"csv":{
						"col_idx": 2,
						"col_hdr": null
					},
					"col_type": "string"
				},
				"col_order_purchase_timestamp": {
					"csv":{
						"col_idx": 3,
						"col_hdr": null,
						"col_format": "2006-01-02 15:04:05"
					},
					"col_type": "datetime"
				}
			}
		}`, srcLine)
	assert.Nil(t, err)

	assert.Equal(t, srcLine[0], colRecord[ReaderAlias]["col_order_id"])
	assert.Equal(t, srcLine[2], colRecord[ReaderAlias]["col_order_status"])
	assert.Equal(t, time.Date(2017, 10, 2, 10, 56, 33, 0, time.UTC), colRecord[ReaderAlias]["col_order_purchase_timestamp"])

	// Good by name
	_, err = testReader(`
		{
			"urls": [""],
			"csv":{
				"hdr_line_idx": 0,
				"first_data_line_idx": 1
			},
			"columns":  {
				"col_order_id": {
					"csv":{
						"col_hdr": "order_id"
					},
					"col_type": "string"
				},
				"col_order_status": {
					"csv":{
						"col_hdr": "order_status"
					},
					"col_type": "string"
				},
				"col_order_purchase_timestamp": {
					"csv":{
						"col_hdr": "order_purchase_timestamp",
						"col_format": "2006-01-02 15:04:05"
					},
					"col_type": "datetime"
				}
			}
		}`, srcLine)
	assert.Nil(t, err)

	// Bad col idx
	_, err = testReader(`
		{
			"urls": [""],
			"csv":{
				"hdr_line_idx": 0,
				"first_data_line_idx": 1
			},
			"columns":  {
				"col_order_id": {
					"csv":{
						"col_idx": -1
					},
					"col_type": "string"
				}
			}
		}`, srcLine)

	assertErrorPrefix(t, "cannot detect csv column indexing mode: [file reader column definition cannot use negative column index: -1]", err.Error())

	// Bad number of source files
	_, err = testReader(`
		{
			"urls": [],
			"csv":{
				"hdr_line_idx": 0,
				"first_data_line_idx": 1
			},
			"columns":  {
				"col_order_id": {
					"csv":{
						"col_hdr": "order_id"
					},
					"col_type": "string"
				},
				"col_order_status": {
					"csv":{
						"col_hdr": "order_status"
					},
					"col_type": "string"
				},
				"col_order_purchase_timestamp": {
					"csv":{
						"col_hdr": "order_purchase_timestamp",
						"col_format": "2006-01-02 15:04:05"
					},
					"col_type": "datetime"
				}
			}
		}`, srcLine)
	assertErrorPrefix(t, "no source file urls specified", err.Error())

	// Bad mixed indexing mode (second column says by idx, first and third say by name)
	_, err = testReader(`
		{
			"urls": [""],
			"csv":{
				"hdr_line_idx": 0,
				"first_data_line_idx": 1
			},
			"columns":  {
				"col_order_id": {
					"csv":{
						"col_idx": 1,
						"col_hdr": "order_id"
					},
					"col_type": "string"
				},
				"col_order_status": {
					"csv":{
						"col_idx": 2
					},
					"col_type": "string"
				},
				"col_order_purchase_timestamp": {
					"csv":{
						"col_hdr": "order_purchase_timestamp",
						"col_format": "2006-01-02 15:04:05"
					},
					"col_type": "datetime"
				}
			}
		}`, srcLine)
	assertErrorPrefix(t, "cannot detect csv column indexing mode", err.Error())

	// Bad: cannot find file header some_unknown_col
	_, err = testReader(`
		{
			"urls": [""],
			"csv":{
				"hdr_line_idx": 0,
				"first_data_line_idx": 1
			},
			"columns":  {
				"col_order_id": {
					"csv":{
						"col_idx": 1,
						"col_hdr": "order_id"
					},
					"col_type": "string"
				},
				"col_order_status": {
					"csv":{
						"col_idx": 2,
						"col_hdr": "some_unknown_col"
					},
					"col_type": "string"
				},
				"col_order_purchase_timestamp": {
					"csv":{
						"col_hdr": "order_purchase_timestamp",
						"col_format": "2006-01-02 15:04:05"
					},
					"col_type": "datetime"
				}
			}
		}`, srcLine)
	assertErrorPrefix(t, "cannot resove all 3 source file column indexes, resolved only 2", err.Error())
}

func TestReadString(t *testing.T) {
	confTemplate := `
	{
		"urls": [""],
		"csv":{
			"hdr_line_idx": 0,
			"first_data_line_idx": 1
		},
		"columns":  {
			"col_1": {
				"csv":{
					%s
					"col_idx": 1
				},
				%s
				"col_type": "string"
			}
		}
	}`

	confNoFormatNoDefault := fmt.Sprintf(confTemplate, ``, ``)
	confNoFormatWithDefault := fmt.Sprintf(confTemplate, ``, `"col_default_value":"default_str",`)
	confWithFormat := fmt.Sprintf(confTemplate, `"col_format": "some_format",`, ``)

	srcLineWithData := []string{"", "data_str", ""}
	srcLineEmpty := []string{"", "", ""}

	goodTestScenarios := [][]any{
		{confNoFormatNoDefault, srcLineWithData, "data_str"},
		{confNoFormatNoDefault, srcLineEmpty, ""},
		{confNoFormatWithDefault, srcLineEmpty, "default_str"},
	}

	for i := 0; i < len(goodTestScenarios); i++ {
		scenario := goodTestScenarios[i]
		colRecord, err := testReader(scenario[0].(string), scenario[1].([]string))
		assert.Nil(t, err)
		assert.Equal(t, scenario[2], colRecord[ReaderAlias]["col_1"], fmt.Sprintf("Test %d", i))
	}

	var err error
	_, err = testReader(confWithFormat, srcLineWithData)
	assertErrorPrefix(t, "cannot read string column col_1, data 'data_str': format 'some_format' was specified, but string fields do not accept format specifier, remove this setting", err.Error())
}

func TestReadDatetime(t *testing.T) {
	confTemplate := `
	{
		"urls": [""],
		"csv":{
			"hdr_line_idx": 0,
			"first_data_line_idx": 1
		},
		"columns":  {
			"col_1": {
				"csv":{
					%s
					"col_idx": 1
				},
				%s
				"col_type": "datetime"
			}
		}
	}`

	confGoodFormatGoodDefault := fmt.Sprintf(confTemplate, `"col_format": "2006-01-02T15:04:05.000",`, `"col_default_value":"2001-07-07T11:22:33.700",`)
	confGoodFormatNoDefault := fmt.Sprintf(confTemplate, `"col_format": "2006-01-02T15:04:05.000",`, ``)
	confNoFormatNoDefault := fmt.Sprintf(confTemplate, ``, ``)
	confNoFormatGoodDefault := fmt.Sprintf(confTemplate, ``, `"col_default_value":"2001-07-07T11:22:33.700",`)
	confGoodFormatBadDefault := fmt.Sprintf(confTemplate, `"col_format": "2006-01-02T15:04:05.000",`, `"col_default_value":"2001-07-07aaa11:22:33.700",`)
	confBadFormatGoodDefault := fmt.Sprintf(confTemplate, `"col_format": "2006-01-02ccc15:04:05.000",`, `"col_default_value":"2001-07-07T11:22:33.700",`)
	confBadFormatBadDefault := fmt.Sprintf(confTemplate, `"col_format": "2006-01-02ccc15:04:05.000",`, `"col_default_value":"2001-07-07aaa11:22:33.700",`)

	srcLineGood := []string{"", "2017-10-02T10:56:33.155"}
	srcLineBad := []string{"", "2017-10-02bbb10:56:33.155"}
	srcLineEmpty := []string{"", ""}

	goodVal := time.Date(2017, time.October, 2, 10, 56, 33, 155000000, time.UTC)
	defaultVal := time.Date(2001, time.July, 7, 11, 22, 33, 700000000, time.UTC)
	nullVal := GetDefaultFieldTypeValue(FieldTypeDateTime)

	goodTestScenarios := [][]any{
		{confGoodFormatGoodDefault, srcLineGood, goodVal},
		{confGoodFormatGoodDefault, srcLineEmpty, defaultVal},
		{confGoodFormatNoDefault, srcLineEmpty, nullVal},
	}

	for i := 0; i < len(goodTestScenarios); i++ {
		scenario := goodTestScenarios[i]
		colRecord, err := testReader(scenario[0].(string), scenario[1].([]string))
		assert.Nil(t, err)
		assert.Equal(t, scenario[2], colRecord[ReaderAlias]["col_1"], fmt.Sprintf("Test %d", i))
	}

	var err error
	_, err = testReader(confNoFormatNoDefault, srcLineGood)
	assertErrorPrefix(t, "cannot read datetime column col_1, data '2017-10-02T10:56:33.155': column format is missing, consider specifying something like 2006-01-02T15:04:05.000-0700, see go datetime format documentation for details", err.Error())
	_, err = testReader(confBadFormatGoodDefault, srcLineGood)
	assertErrorPrefix(t, `cannot read datetime column col_1, data '2017-10-02T10:56:33.155', format '2006-01-02ccc15:04:05.000': parsing time "2017-10-02T10:56:33.155" as "2006-01-02ccc15:04:05.000": cannot parse "T10:56:33.155" as "ccc"`, err.Error())
	_, err = testReader(confBadFormatBadDefault, srcLineEmpty)
	assertErrorPrefix(t, `cannot read time column col_1 from default value string '2001-07-07aaa11:22:33.700': parsing time "2001-07-07aaa11:22:33.700" as "2006-01-02ccc15:04:05.000": cannot parse "aaa11:22:33.700" as "ccc"`, err.Error())
	_, err = testReader(confNoFormatGoodDefault, srcLineEmpty)
	assertErrorPrefix(t, `cannot read time column col_1 from default value string '2001-07-07T11:22:33.700': parsing time "2001-07-07T11:22:33.700": extra text: "2001-07-07T11:22:33.700"`, err.Error())
	_, err = testReader(confGoodFormatBadDefault, srcLineEmpty)
	assertErrorPrefix(t, `cannot read time column col_1 from default value string '2001-07-07aaa11:22:33.700': parsing time "2001-07-07aaa11:22:33.700" as "2006-01-02T15:04:05.000": cannot parse "aaa11:22:33.700" as "T"`, err.Error())
	_, err = testReader(confGoodFormatGoodDefault, srcLineBad)
	assertErrorPrefix(t, `cannot read datetime column col_1, data '2017-10-02bbb10:56:33.155', format '2006-01-02T15:04:05.000': parsing time "2017-10-02bbb10:56:33.155" as "2006-01-02T15:04:05.000": cannot parse "bbb10:56:33.155" as "T"`, err.Error())
}

func TestReadInt(t *testing.T) {
	confTemplate := `
	{
		"urls": [""],
		"csv":{
			"hdr_line_idx": 0,
			"first_data_line_idx": 1
		},
		"columns":  {
			"col_1": {
				"csv":{
					%s
					"col_idx": 1
				},
				%s
				"col_type": "int"
			}
		}
	}`

	confComplexFormatWithDefault := fmt.Sprintf(confTemplate, `"col_format": "value(%d)",`, `"col_default_value":"123",`)
	confSimpleFormatNoDefault := fmt.Sprintf(confTemplate, `"col_format": "%d",`, ``)
	confNoFormatNoDefault := fmt.Sprintf(confTemplate, ``, ``)
	confNoFormatBadDefault := fmt.Sprintf(confTemplate, ``, `"col_default_value":"badstring",`)

	srcLineComplexFormat := []string{"", "value(111)", ""}
	srcLineSimpleFormat := []string{"", "111", ""}
	srcLineEmpty := []string{"", "", ""}

	goodTestScenarios := [][]any{
		{confComplexFormatWithDefault, srcLineComplexFormat, int64(111)},
		{confSimpleFormatNoDefault, srcLineSimpleFormat, int64(111)},
		{confNoFormatNoDefault, srcLineSimpleFormat, int64(111)},
		{confComplexFormatWithDefault, srcLineEmpty, int64(123)},
		{confSimpleFormatNoDefault, srcLineEmpty, int64(0)},
	}

	for i := 0; i < len(goodTestScenarios); i++ {
		scenario := goodTestScenarios[i]
		colRecord, err := testReader(scenario[0].(string), scenario[1].([]string))
		assert.Nil(t, err)
		assert.Equal(t, scenario[2], colRecord[ReaderAlias]["col_1"], fmt.Sprintf("Test %d", i))
	}

	var err error
	_, err = testReader(confSimpleFormatNoDefault, srcLineComplexFormat)
	assertErrorPrefix(t, "cannot read int64 column col_1, data 'value(111)', format '%d': expected integer", err.Error())
	_, err = testReader(confComplexFormatWithDefault, srcLineSimpleFormat)
	assertErrorPrefix(t, "cannot read int64 column col_1, data '111', format 'value(%d)': input does not match format", err.Error())
	_, err = testReader(confNoFormatBadDefault, srcLineEmpty)
	assertErrorPrefix(t, `cannot read int64 column col_1 from default value string 'badstring': strconv.ParseInt: parsing "badstring": invalid syntax`, err.Error())
	_, err = testReader(confNoFormatBadDefault, srcLineComplexFormat)
	assertErrorPrefix(t, `cannot read int64 column col_1, data 'value(111)', no format: strconv.ParseInt: parsing "value(111)": invalid syntax`, err.Error())
}

func TestReadFloat(t *testing.T) {
	confTemplate := `
	{
		"urls": [""],
		"csv":{
			"hdr_line_idx": 0,
			"first_data_line_idx": 1
		},
		"columns":  {
			"col_1": {
				"csv":{
					%s
					"col_idx": 1
				},
				%s
				"col_type": "float"
			}
		}
	}`

	confComplexFormatWithDefault := fmt.Sprintf(confTemplate, `"col_format": "value(%f)",`, `"col_default_value":"5.697",`)
	confSimpleFormatNoDefault := fmt.Sprintf(confTemplate, `"col_format": "%f",`, ``)
	confNoFormatNoDefault := fmt.Sprintf(confTemplate, ``, ``)
	confNoFormatBadDefault := fmt.Sprintf(confTemplate, ``, `"col_default_value":"badstring",`)

	srcLineComplexFormat := []string{"", "value(111.222)", ""}
	srcLineSimpleFormat := []string{"", "111.222", ""}
	srcLineEmpty := []string{"", "", ""}

	goodTestScenarios := [][]any{
		{confComplexFormatWithDefault, srcLineComplexFormat, float64(111.222)},
		{confSimpleFormatNoDefault, srcLineSimpleFormat, float64(111.222)},
		{confNoFormatNoDefault, srcLineSimpleFormat, float64(111.222)},
		{confComplexFormatWithDefault, srcLineEmpty, float64(5.697)},
		{confSimpleFormatNoDefault, srcLineEmpty, float64(0.0)},
	}

	for i := 0; i < len(goodTestScenarios); i++ {
		scenario := goodTestScenarios[i]
		colRecord, err := testReader(scenario[0].(string), scenario[1].([]string))
		assert.Nil(t, err)
		assert.Equal(t, scenario[2], colRecord[ReaderAlias]["col_1"], fmt.Sprintf("Test %d", i))
	}

	var err error
	_, err = testReader(confSimpleFormatNoDefault, srcLineComplexFormat)
	assertErrorPrefix(t, `cannot read float64 column col_1, data 'value(111.222)', format '%f': strconv.ParseFloat: parsing "": invalid syntax`, err.Error())
	_, err = testReader(confComplexFormatWithDefault, srcLineSimpleFormat)
	assertErrorPrefix(t, "cannot read float64 column col_1, data '111.222', format 'value(%f)': input does not match format", err.Error())
	_, err = testReader(confNoFormatBadDefault, srcLineEmpty)
	assertErrorPrefix(t, `cannot read float64 column col_1 from default value string 'badstring': strconv.ParseFloat: parsing "badstring": invalid syntax`, err.Error())
	_, err = testReader(confNoFormatBadDefault, srcLineComplexFormat)
	assertErrorPrefix(t, `cannot read float64 column col_1, data 'value(111.222)', no format: strconv.ParseFloat: parsing "value(111.222)": invalid syntax`, err.Error())
}

func TestReadDecimal(t *testing.T) {
	confTemplate := `
	{
		"urls": [""],
		"csv":{
			"hdr_line_idx": 0,
			"first_data_line_idx": 1
		},
		"columns":  {
			"col_1": {
				"csv":{
					%s
					"col_idx": 1
				},
				%s
				"col_type": "decimal2"
			}
		}
	}`

	confComplexFormatWithDefault := fmt.Sprintf(confTemplate, `"col_format": "value(%f)",`, `"col_default_value":"-56.78",`)
	confSimpleFormatNoDefault := fmt.Sprintf(confTemplate, `"col_format": "%f",`, ``)
	confNoFormatNoDefault := fmt.Sprintf(confTemplate, ``, ``)
	confNoFormatBadDefault := fmt.Sprintf(confTemplate, ``, `"col_default_value":"badstring",`)

	srcLineComplexFormat := []string{"", "value(12.34)", ""}
	srcLineSimpleFormat := []string{"", "12.34", ""}
	srcLineEmpty := []string{"", "", ""}

	goodTestScenarios := [][]any{
		{confComplexFormatWithDefault, srcLineComplexFormat, decimal.NewFromFloat32(12.34)},
		{confSimpleFormatNoDefault, srcLineSimpleFormat, decimal.NewFromFloat32(12.34)},
		{confNoFormatNoDefault, srcLineSimpleFormat, decimal.NewFromFloat32(12.34)},
		{confComplexFormatWithDefault, srcLineEmpty, decimal.NewFromFloat32(-56.78)},
		{confSimpleFormatNoDefault, srcLineEmpty, decimal.NewFromFloat32(0.0)},
	}

	for i := 0; i < len(goodTestScenarios); i++ {
		scenario := goodTestScenarios[i]
		colRecord, err := testReader(scenario[0].(string), scenario[1].([]string))
		assert.Nil(t, err)
		assert.Equal(t, scenario[2], colRecord[ReaderAlias]["col_1"], fmt.Sprintf("Test %d", i))
	}

	var err error
	_, err = testReader(confSimpleFormatNoDefault, srcLineComplexFormat)
	assertErrorPrefix(t, `cannot read decimal2 column col_1, data 'value(12.34)', format '%f': strconv.ParseFloat: parsing "": invalid syntax`, err.Error())
	_, err = testReader(confComplexFormatWithDefault, srcLineSimpleFormat)
	assertErrorPrefix(t, "cannot read decimal2 column col_1, data '12.34', format 'value(%f)': input does not match format", err.Error())
	_, err = testReader(confNoFormatBadDefault, srcLineEmpty)
	assertErrorPrefix(t, `cannot read decimal2 column col_1 from default value string 'badstring': can't convert badstring to decimal`, err.Error())
	_, err = testReader(confNoFormatBadDefault, srcLineComplexFormat)
	assertErrorPrefix(t, `cannot read decimal2 column col_1, cannot parse data 'value(12.34)': can't convert value(12.34) to decimal: exponent is not numeric`, err.Error())
}
func TestReadBool(t *testing.T) {

	confTemplate := `
	{
		"urls": [""],
		"csv":{
			"hdr_line_idx": 0,
			"first_data_line_idx": 1
		},
		"columns":  {
			"col_1": {
				"csv":{
					%s
					"col_idx": 1
				},
				%s
				"col_type": "bool"
			}
		}
	}`

	confNoFormatNoDefault := fmt.Sprintf(confTemplate, ``, ``)
	confNoFormatWithDefault := fmt.Sprintf(confTemplate, ``, `"col_default_value":"TRUE",`)
	confNoFormatBadDefault := fmt.Sprintf(confTemplate, ``, `"col_default_value":"baddefault",`)
	confWithFormat := fmt.Sprintf(confTemplate, `"col_format": "some_format",`, ``)

	srcLineTrue := []string{"", "True", ""}
	srcLineFalse := []string{"", "False", ""}
	srcLineTrueCap := []string{"", "TRUE", ""}
	srcLineFalseCap := []string{"", "FALSE", ""}
	srcLineTrueSmall := []string{"", "true", ""}
	srcLineFalseSmall := []string{"", "false", ""}
	srcLineT := []string{"", "T", ""}
	srcLineF := []string{"", "F", ""}
	srcLineTSmall := []string{"", "t", ""}
	srcLineFSmall := []string{"", "f", ""}
	srcLine0 := []string{"", "0", ""}
	srcLine1 := []string{"", "1", ""}
	srcLineEmpty := []string{"", "", ""}
	srcLineBad := []string{"", "bad", ""}

	goodTestScenarios := [][]any{
		{confNoFormatNoDefault, srcLineTrue, true},
		{confNoFormatNoDefault, srcLineFalse, false},
		{confNoFormatNoDefault, srcLineTrueCap, true},
		{confNoFormatNoDefault, srcLineFalseCap, false},
		{confNoFormatNoDefault, srcLineTrueSmall, true},
		{confNoFormatNoDefault, srcLineFalseSmall, false},
		{confNoFormatNoDefault, srcLineT, true},
		{confNoFormatNoDefault, srcLineF, false},
		{confNoFormatNoDefault, srcLineTSmall, true},
		{confNoFormatNoDefault, srcLineFSmall, false},
		{confNoFormatNoDefault, srcLine1, true},
		{confNoFormatNoDefault, srcLine0, false},
		{confNoFormatWithDefault, srcLineEmpty, true},
		{confNoFormatNoDefault, srcLineEmpty, false},
	}

	for i := 0; i < len(goodTestScenarios); i++ {
		scenario := goodTestScenarios[i]
		colRecord, err := testReader(scenario[0].(string), scenario[1].([]string))
		assert.Nil(t, err)
		assert.Equal(t, scenario[2], colRecord[ReaderAlias]["col_1"], fmt.Sprintf("Test %d", i))
	}

	var err error
	_, err = testReader(confNoFormatNoDefault, srcLineBad)
	assertErrorPrefix(t, `cannot read bool column col_1, data 'bad', allowed values are true,false,T,F,0,1: strconv.ParseBool: parsing "bad": invalid syntax`, err.Error())
	_, err = testReader(confNoFormatBadDefault, srcLineEmpty)
	assertErrorPrefix(t, `cannot read bool column col_1, from default value string 'baddefault', allowed values are true,false,T,F,0,1: strconv.ParseBool: parsing "baddefault": invalid syntax`, err.Error())
	_, err = testReader(confWithFormat, srcLineTrue)
	assertErrorPrefix(t, `cannot read bool column col_1, data 'True': format 'some_format' was specified, but bool fields do not accept format specifier, remove this setting`, err.Error())
}
