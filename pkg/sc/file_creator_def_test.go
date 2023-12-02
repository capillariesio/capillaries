package sc

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

const nodeCfgCsvJson string = `
{
	"top": {
		"order": "field_string1(asc)"
	},
	"url_template": "taxed_table1.csv",
	"columns": [
		{
			"csv":{
				"header": "field_string1",
				"format": "%s"
			},
			"name": "field_string1",
			"expression": "r.field_string1",
			"type": "string"
		}
	]
}
`

const nodeCfgParquetJson string = `
{
	"url_template": "taxed_table1.csv",
	"columns": [
		{
			"parquet":{
				"column_name": "field_string1"
			},
			"name": "field_string1",
			"expression": "r.field_string1",
			"type": "string"
		}
	]
}
`

const nodeCfgJsonNoColumns string = `
{
	"url_template": "out_file.csv",
	"columns": []
}
`

func TestFileCreatorDefFailures(t *testing.T) {
	c := FileCreatorDef{}

	assert.Nil(t, c.Deserialize([]byte(nodeCfgCsvJson)))
	assert.Nil(t, c.Deserialize([]byte(nodeCfgParquetJson)))
	assert.Contains(t, c.Deserialize([]byte(nodeCfgJsonNoColumns)).Error(), "cannot cannot detect file creator type: parquet should have column_name, csv should have header")

	re := regexp.MustCompile(`"type": "[^"]+"`)
	assert.Contains(t, c.Deserialize([]byte(re.ReplaceAllString(nodeCfgCsvJson, `"type": "aaa"`))).Error(), "invalid column type [aaa]")

	re = regexp.MustCompile(`"order": "[^"]+"`)
	assert.Contains(t, c.Deserialize([]byte(re.ReplaceAllString(nodeCfgCsvJson, `"order": "bad_field(asc)"`))).Error(), "cannot parse raw index definition(s) for top")

}
