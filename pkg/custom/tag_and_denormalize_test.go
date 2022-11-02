package custom

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	"github.com/capillariesio/capillaries/pkg/sc"
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

func TestTagAndDenormalizeDef(t *testing.T) {
	script := `
	{
		"nodes": {
			"read_products": {
				"type": "file_table",
				"desc": "Load product data from CSV files to a table, one input file - one batch",
				"explicit_run_only": true,
				"r": {
					"urls": ["{test_root_dir}/data/in/flipcart_products.tsv"],
					"separator": "\t",
					"hdr_line_idx": 0,
					"first_data_line_idx": 1,
					"columns": {
						"col_product_id": {
							"col_idx": 0,
							"col_format": "%d",
							"col_type": "int"
						},
						"col_product_name": {
							"col_idx": 1,
							"col_type": "string"
						},
						"col_product_category_tree": {
							"col_idx": 2,
							"col_type": "string"
						},
						"col_retail_price": {
							"col_idx": 3,
							"col_format": "%f",
							"col_type": "decimal2"
						},
						"col_product_specifications": {
							"col_idx": 4,
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
						"idx_tagged_products_tag": "non_unique(tag)"
					}
				}
			}
		},
		"dependency_policies": {
			"current_active_first_stopped_nogo": {
				"is_default": true,
				"event_priority_order": "run_is_current(desc), node_start_ts(desc)",
				"rules": [
					{
						"cmd": "go",
						"expression": "e.run_is_current == true && e.run_status == wfmodel.RunStart && e.node_status == wfmodel.NodeBatchSuccess"
					},
					{
						"cmd": "wait",
						"expression": "e.run_is_current == true && e.run_status == wfmodel.RunStart && e.node_status == wfmodel.NodeBatchNone"
					},
					{
						"cmd": "wait",
						"expression": "e.run_is_current == true && e.run_status == wfmodel.RunStart && e.node_status == wfmodel.NodeBatchStart"
					},
					{
						"cmd": "nogo",
						"expression": "e.run_is_current == true && e.run_status == wfmodel.RunStart && e.node_status == wfmodel.NodeBatchFail"
					},
					{
						"cmd": "go",
						"expression": "e.run_is_current == false && e.run_status == wfmodel.RunStart && e.node_status == wfmodel.NodeBatchSuccess"
					},
					{
						"cmd": "wait",
						"expression": "e.run_is_current == false && e.run_status == wfmodel.RunStart && e.node_status == wfmodel.NodeBatchNone"
					},
					{
						"cmd": "wait",
						"expression": "e.run_is_current == false && e.run_status == wfmodel.RunStart && e.node_status == wfmodel.NodeBatchStart"
					},
					{
						"cmd": "go",
						"expression": "e.run_is_current == false && e.run_status == wfmodel.RunComplete && e.node_status == wfmodel.NodeBatchSuccess"
					},
					{
						"cmd": "nogo",
						"expression": "e.run_is_current == false && e.run_status == wfmodel.RunComplete && e.node_status == wfmodel.NodeBatchFail"
					}
				]
			}
		}
	}`

	var err error
	var tndProcessor *TagAndDenormalizeProcessorDef

	newScript := &sc.ScriptDef{}
	if err = newScript.Deserialize([]byte(script), &TagAndDenormalizeTestTestProcessorDefFactory{}, map[string]json.RawMessage{"tag_and_denormalize": {}}, ""); err != nil {
		t.Error(err)
	}
	tndProcessor, _ = newScript.ScriptNodes["tag_products"].CustomProcessor.(*TagAndDenormalizeProcessorDef)
	assert.Equal(t, 3, len(tndProcessor.ParsedTagCriteria))

	re := regexp.MustCompile(`"tag_criteria": \{[^\}]+\}`)
	if err = newScript.Deserialize(
		[]byte(re.ReplaceAllString(script, `"tag_criteria_uri": "../../test/data/cfg/tag_and_denormalize/tag_criteria.json"`)),
		&TagAndDenormalizeTestTestProcessorDefFactory{}, map[string]json.RawMessage{"tag_and_denormalize": {}}, ""); err != nil {
		t.Error(err)
	}
	tndProcessor, _ = newScript.ScriptNodes["tag_products"].CustomProcessor.(*TagAndDenormalizeProcessorDef)
	assert.Equal(t, 4, len(tndProcessor.ParsedTagCriteria))

	err = newScript.Deserialize(
		[]byte(strings.Replace(script, `"having": "len(w.tag) > 0"`, `"having": "len(p.tag) > 0"`, 1)),
		&TagAndDenormalizeTestTestProcessorDefFactory{}, map[string]json.RawMessage{"tag_and_denormalize": {}}, "")
	assert.Contains(t, err.Error(), "prohibited field p.tag")

	// Exercise checkFieldUsageInCustomProcessor() error code path

	err = newScript.Deserialize(
		[]byte(strings.ReplaceAll(script, `r.product_spec`, `w.product_spec`)),
		&TagAndDenormalizeTestTestProcessorDefFactory{}, map[string]json.RawMessage{"tag_and_denormalize": {}}, "")
	assert.Contains(t, err.Error(), "unknown field w.product_spec")

}
