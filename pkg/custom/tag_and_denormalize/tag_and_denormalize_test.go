package tag_and_denormalize

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/capillariesio/capillaries/pkg/eval"
	"github.com/capillariesio/capillaries/pkg/proc"
	"github.com/capillariesio/capillaries/pkg/sc"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

type TagAndDenormalizeTestTestProcessorDefFactory struct {
}

func (f *TagAndDenormalizeTestTestProcessorDefFactory) Create(processorType string) (sc.CustomProcessorDef, bool) {
	switch processorType {
	case ProcessorTagAndDenormalizeName:
		return &TagAndDenormalizeProcessorDef{}, true
	default:
		return nil, false
	}
}

const scriptJson string = `
{
	"nodes": {
		"read_products": {
			"type": "file_table",
			"desc": "Load product data from CSV files to a table, one input file - one batch",
			"explicit_run_only": true,
			"r": {
				"urls": ["{test_root_dir}/data/in/flipcart_products.tsv"],
				"csv":{
					"separator": "\t",
					"hdr_line_idx": 0,
					"first_data_line_idx": 1
				},
				"columns": {
					"col_product_id": {
						"csv":{
							"col_idx": 0,
							"col_format": "%d"
						},
						"col_type": "int"
					},
					"col_product_name": {
						"csv":{
							"col_idx": 1
						},
						"col_type": "string"
					},
					"col_product_category_tree": {
						"csv":{
							"col_idx": 2
						},
						"col_type": "string"
					},
					"col_retail_price": {
						"csv":{
							"col_idx": 3,
							"col_format": "%f"
						},
						"col_type": "decimal2"
					},
					"col_product_specifications": {
						"csv":{
							"col_idx": 4
						},
						"col_type": "string"
					}
				}
			},
			"w": {
				"name": "products",
				"fields": {
					"product_id": {
						"expression": "r.col_product_id",
						"type": "int"
					},
					"name": {
						"expression": "r.col_product_name",
						"type": "string"
					},
					"category_tree": {
						"expression": "r.col_product_category_tree",
						"type": "string"
					},
					"price": {
						"expression": "r.col_retail_price",
						"type": "decimal2"
					},
					"product_spec": {
						"expression": "r.col_product_specifications",
						"type": "string"
					}
				}
			}
		},
		"tag_products": {
			"type": "table_custom_tfm_table",
			"custom_proc_type": "tag_and_denormalize",
			"desc": "Tag products according to criteria and write product tag, id, price to a new table",
			"r": {
				"table": "products",
				"expected_batches_total": 10
			},
			"p": {
				"tag_field_name": "tag",
				"tag_criteria": {
					"boys":"re.MatchString(` + "`" + `\"k\":\"Ideal For\",\"v\":\"[\\w ,]*Boys[\\w ,]*\"` + "`" + `, r.product_spec)",
					"diving":"re.MatchString(` + "`" + `\"k\":\"Water Resistance Depth\",\"v\":\"(100|200) m\"` + "`" + `, r.product_spec)",
					"engagement":"re.MatchString(` + "`" + `\"k\":\"Occasion\",\"v\":\"[\\w ,]*Engagement[\\w ,]*\"` + "`" + `, r.product_spec) && re.MatchString(` + "`" + `\"k\":\"Gemstone\",\"v\":\"Diamond\"` + "`" + `, r.product_spec) && r.price > 5000"
				}
			},
			"w": {
				"name": "tagged_products",
				"having": "len(w.tag) > 0",
				"fields": {
					"tag": {
						"expression": "p.tag",
						"type": "string"
					},
					"product_id": {
						"expression": "r.product_id",
						"type": "int"
					},
					"price": {
						"expression": "r.price",
						"type": "decimal2"
					}
				},
				"indexes": {
				}
			}
		}
	},
	"dependency_policies": {
		"current_active_first_stopped_nogo":` + sc.DefaultPolicyCheckerConf +
	`		
	}
}`

func TestTagAndDenormalizeDeserializeFileCriteria(t *testing.T) {
	scriptDef := &sc.ScriptDef{}

	re := regexp.MustCompile(`"tag_criteria": \{[^\}]+\}`)
	err := scriptDef.Deserialize(
		[]byte(re.ReplaceAllString(scriptJson, `"tag_criteria_uri": "../../../test/data/cfg/tag_and_denormalize_quicktest/tag_criteria.json"`)),
		&TagAndDenormalizeTestTestProcessorDefFactory{}, map[string]json.RawMessage{"tag_and_denormalize": {}}, "", nil)
	assert.Nil(t, err)

	tndProcessor, ok := scriptDef.ScriptNodes["tag_products"].CustomProcessor.(*TagAndDenormalizeProcessorDef)
	assert.True(t, ok)
	assert.Equal(t, 4, len(tndProcessor.ParsedTagCriteria))
}

func TestTagAndDenormalizeRunEmbeddedCriteria(t *testing.T) {
	scriptDef := &sc.ScriptDef{}

	err := scriptDef.Deserialize([]byte(scriptJson), &TagAndDenormalizeTestTestProcessorDefFactory{}, map[string]json.RawMessage{"tag_and_denormalize": {}}, "", nil)
	assert.Nil(t, err)

	tndProcessor, ok := scriptDef.ScriptNodes["tag_products"].CustomProcessor.(*TagAndDenormalizeProcessorDef)
	assert.True(t, ok)
	assert.Equal(t, 3, len(tndProcessor.ParsedTagCriteria))

	// Initializing rowset is tedious and error-prone. Add schema first.
	rs := proc.NewRowsetFromFieldRefs(sc.FieldRefs{
		{TableName: "r", FieldName: "product_id", FieldType: sc.FieldTypeInt},
		{TableName: "r", FieldName: "name", FieldType: sc.FieldTypeString},
		{TableName: "r", FieldName: "price", FieldType: sc.FieldTypeDecimal2},
		{TableName: "r", FieldName: "product_spec", FieldType: sc.FieldTypeString},
	})

	// Allocate rows
	assert.Nil(t, rs.InitRows(1))

	// Initialize with pointers
	product_id := int64(1)
	(*rs.Rows[0])[0] = &product_id
	name := "Breitling AB011010/BB08 131S Chronomat 44 Analog Watch"
	(*rs.Rows[0])[1] = &name
	price := decimal.NewFromFloat(571230)
	(*rs.Rows[0])[2] = &price
	product_spec := `{"k":"Occasion","v":"Formal, Casual"}, {"k":"Ideal For","v":"Boys, Men"}, {"k":"Water Resistance Depth","v":"100 m"}`
	(*rs.Rows[0])[3] = &product_spec

	// Tell it we wrote something to [0]
	rs.RowCount++

	// Test flusher, doesn't write anywhere, just saves data in the local variable
	var results []*eval.VarValuesMap
	flushVarsArray := func(varsArray []*eval.VarValuesMap, _ int) error {
		results = varsArray
		return nil
	}

	err = tndProcessor.tagAndDenormalize(rs, flushVarsArray)
	assert.Nil(t, err)

	// Check that 2 rows were produced: thiswatch is good for boys and for diving

	flushedRow := *results[0]
	// r fields must be present in the result, they can be used by the writer
	assert.Equal(t, product_id, flushedRow["r"]["product_id"])
	assert.Equal(t, name, flushedRow["r"]["name"])
	assert.Equal(t, price, flushedRow["r"]["price"])
	assert.Equal(t, product_spec, flushedRow["r"]["product_spec"])
	// p field must be in the result
	var nextExpectedTag string
	flushedRowTag, ok := flushedRow["p"]["tag"].(string)
	assert.True(t, ok)
	if flushedRowTag == "boys" {
		nextExpectedTag = "diving"
	} else if flushedRowTag == "diving" {
		nextExpectedTag = "boys"
	} else {
		assert.Fail(t, fmt.Sprintf("unexpected tag %s", flushedRowTag))
	}

	flushedRow = *results[1]
	// r fields must be present in the result, they can be used by the writer
	assert.Equal(t, product_id, flushedRow["r"]["product_id"])
	assert.Equal(t, name, flushedRow["r"]["name"])
	assert.Equal(t, price, flushedRow["r"]["price"])
	assert.Equal(t, product_spec, flushedRow["r"]["product_spec"])
	// p field must be in the result
	assert.Equal(t, nextExpectedTag, flushedRow["p"]["tag"])

	// Bad criteria
	re := regexp.MustCompile(`"tag_criteria": \{[^\}]+\}`)

	// Bad function used
	assert.Nil(t, scriptDef.Deserialize(
		[]byte(re.ReplaceAllString(scriptJson, `"tag_criteria": {"boys":"re.BadGoMethod(\"aaa\")"}`)),
		&TagAndDenormalizeTestTestProcessorDefFactory{}, map[string]json.RawMessage{"tag_and_denormalize": {}}, "", nil))

	tndProcessor, ok = scriptDef.ScriptNodes["tag_products"].CustomProcessor.(*TagAndDenormalizeProcessorDef)
	assert.True(t, ok)
	assert.Equal(t, 1, len(tndProcessor.ParsedTagCriteria))

	err = tndProcessor.tagAndDenormalize(rs, flushVarsArray)
	assert.Contains(t, err.Error(), "cannot evaluate expression for tag boys criteria")

	// Bad type
	assert.Nil(t, scriptDef.Deserialize(
		[]byte(re.ReplaceAllString(scriptJson, `"tag_criteria": {"boys":"math.Round(1.1)"}`)),
		&TagAndDenormalizeTestTestProcessorDefFactory{}, map[string]json.RawMessage{"tag_and_denormalize": {}}, "", nil))
	tndProcessor, ok = scriptDef.ScriptNodes["tag_products"].CustomProcessor.(*TagAndDenormalizeProcessorDef)
	assert.True(t, ok)
	assert.Equal(t, 1, len(tndProcessor.ParsedTagCriteria))

	err = tndProcessor.tagAndDenormalize(rs, flushVarsArray)
	assert.Contains(t, err.Error(), "tag boys criteria returned type float64, expected bool")
}

func TestTagAndDenormalizeDeserializeFailures(t *testing.T) {
	scriptDef := &sc.ScriptDef{}

	// Exercise checkFieldUsageInCustomProcessor() error code path
	err := scriptDef.Deserialize(
		[]byte(strings.ReplaceAll(scriptJson, `r.product_spec`, `w.product_spec`)),
		&TagAndDenormalizeTestTestProcessorDefFactory{}, map[string]json.RawMessage{"tag_and_denormalize": {}}, "", nil)
	assert.Contains(t, err.Error(), "unknown field w.product_spec")

	// Prohibited field
	err = scriptDef.Deserialize(
		[]byte(strings.Replace(scriptJson, `"having": "len(w.tag) > 0"`, `"having": "len(p.tag) > 0"`, 1)),
		&TagAndDenormalizeTestTestProcessorDefFactory{}, map[string]json.RawMessage{"tag_and_denormalize": {}}, "", nil)
	assert.Contains(t, err.Error(), "prohibited field p.tag")

	// Bad criteria
	re := regexp.MustCompile(`"tag_criteria": \{[^\}]+\}`)
	err = scriptDef.Deserialize(
		[]byte(re.ReplaceAllString(scriptJson, `"some_bogus_key": 123`)),
		&TagAndDenormalizeTestTestProcessorDefFactory{}, map[string]json.RawMessage{"tag_and_denormalize": {}}, "", nil)
	assert.Contains(t, err.Error(), "cannot unmarshal with tag_criteria and tag_criteria_url missing")

	err = scriptDef.Deserialize(
		[]byte(re.ReplaceAllString(scriptJson, `"tag_criteria":{"a":"b"},"tag_criteria_uri":"aaa"`)),
		&TagAndDenormalizeTestTestProcessorDefFactory{}, map[string]json.RawMessage{"tag_and_denormalize": {}}, "", nil)
	assert.Contains(t, err.Error(), "cannot unmarshal both tag_criteria and tag_criteria_url - pick one")

	err = scriptDef.Deserialize(
		[]byte(re.ReplaceAllString(scriptJson, `"tag_criteria_uri":"aaa"`)),
		&TagAndDenormalizeTestTestProcessorDefFactory{}, map[string]json.RawMessage{"tag_and_denormalize": {}}, "", nil)
	assert.Contains(t, err.Error(), "cannot get criteria file")

	err = scriptDef.Deserialize(
		[]byte(re.ReplaceAllString(scriptJson, `"tag_criteria": ["boys"]`)),
		&TagAndDenormalizeTestTestProcessorDefFactory{}, map[string]json.RawMessage{"tag_and_denormalize": {}}, "", nil)
	assert.Contains(t, err.Error(), "cannot unmarshal array into Go struct")

	err = scriptDef.Deserialize(
		[]byte(re.ReplaceAllString(scriptJson, `"tag_criteria": {"boys":"["}`)),
		&TagAndDenormalizeTestTestProcessorDefFactory{}, map[string]json.RawMessage{"tag_and_denormalize": {}}, "", nil)
	assert.Contains(t, err.Error(), "cannot parse tag criteria expression")
}
