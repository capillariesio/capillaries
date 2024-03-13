package sc

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

const scriptDefJson string = `
{
    "nodes": {
        "read_orders": {
            "type": "file_table",
            "r": {
                "urls": [
                    "{dir_in}/olist_orders_dataset.csv"
                ],
                "csv": {
                    "hdr_line_idx": 0,
                    "first_data_line_idx": 1
                },
                "columns": {
                    "col_order_id": {
                        "csv": {
                            "col_hdr": "order_id"
                        },
                        "col_type": "string"
                    },
                    "col_order_status": {
                        "csv": {
                            "col_hdr": "order_status"
                        },
                        "col_type": "string"
                    },
                    "col_order_purchase_timestamp": {
                        "csv": {
                            "col_hdr": "order_purchase_timestamp",
                            "col_format": "2006-01-02 15:04:05"
                        },
                        "col_type": "datetime"
                    }
                }
            },
            "w": {
                "name": "orders",
                "fields": {
                    "order_id": {
                        "expression": "r.col_order_id",
                        "type": "string"
                    },
                    "order_status": {
                        "expression": "r.col_order_status",
                        "type": "string"
                    },
                    "order_purchase_timestamp": {
                        "expression": "r.col_order_purchase_timestamp",
                        "type": "datetime"
                    }
                }
            }
        },
        "read_order_items": {
            "type": "file_table",
            "r": {
                "urls": [
                    "{dir_in}/olist_order_items_dataset.csv"
                ],
                "csv":{
                    "hdr_line_idx": 0,
                    "first_data_line_idx": 1
                },
                "columns": {
                    "col_order_id": {
                        "csv": {
                            "col_idx": 0,
                            "col_hdr": null
                        },
                        "col_type": "string"
                    },
                    "col_order_item_id": {
                        "csv": {
                            "col_idx": 1,
                            "col_hdr": null,
                            "col_format": "%d"
                        },
                        "col_type": "int"
                    },
                    "col_product_id": {
                        "csv": {
                            "col_idx": 2,
                            "col_hdr": null
                        },
                        "col_type": "string"
                    },
                    "col_seller_id": {
                        "csv": {
                            "col_idx": 3,
                            "col_hdr": null
                        },
                        "col_type": "string"
                    },
                    "col_shipping_limit_date": {
                        "csv": {
                            "col_idx": 4,
                            "col_hdr": null,
                            "col_format": "2006-01-02 15:04:05"
                        },
                        "col_type": "datetime"
                    },
                    "col_price": {
                        "csv": {
                            "col_idx": 5,
                            "col_hdr": null,
                            "col_format": "%f"
                        },
                        "col_type": "decimal2"
                    },
                    "col_freight_value": {
                        "csv": {
                            "col_idx": 6,
                            "col_hdr": null,
                            "col_format": "%f"
                        },
                        "col_type": "decimal2"
                    }
                }
            },
            "w": {
                "name": "order_items",
                "having": null,
                "fields": {
                    "order_id": {
                        "expression": "r.col_order_id",
                        "type": "string"
                    },
                    "order_item_id": {
                        "expression": "r.col_order_item_id",
                        "type": "int"
                    },
                    "product_id": {
                        "expression": "r.col_product_id",
                        "type": "string"
                    },
                    "seller_id": {
                        "expression": "r.col_seller_id",
                        "type": "string"
                    },
                    "shipping_limit_date": {
                        "expression": "r.col_shipping_limit_date",
                        "type": "datetime"
                    },
                    "value": {
                        "expression": "r.col_price+r.col_freight_value",
                        "type": "decimal2"
                    }
                },
                "indexes": {
                    "idx_order_items_order_id": "non_unique(order_id(case_sensitive))"
                }
            }
        },
        "order_item_date_inner": {
            "type": "table_lookup_table",
            "r": {
                "table": "orders",
                "expected_batches_total": 100
            },
            "l": {
                "index_name": "idx_order_items_order_id",
				"idx_read_batch_size": 3000,
				"right_lookup_read_batch_size": 5000,
				"filter": "len(l.product_id) > 0",
                "join_on": "r.order_id",
                "group": false,
                "join_type": "inner"
            },
            "w": {
                "name": "order_item_date_inner",
                "fields": {
                    "order_id": {
                        "expression": "r.order_id",
                        "type": "string"
                    },
                    "order_purchase_timestamp": {
                        "expression": "r.order_purchase_timestamp",
                        "type": "datetime"
                    },
                    "order_item_id": {
                        "expression": "l.order_item_id",
                        "type": "int"
                    },
                    "product_id": {
                        "expression": "l.product_id",
                        "type": "string"
                    },
                    "seller_id": {
                        "expression": "l.seller_id",
                        "type": "string"
                    },
                    "shipping_limit_date": {
                        "expression": "l.shipping_limit_date",
                        "type": "datetime"
                    },
                    "value": {
                        "expression": "l.value",
                        "type": "decimal2"
                    }
                }
            }
        }
	},
	"dependency_policies": {
		"current_active_first_stopped_nogo":` + DefaultPolicyCheckerConf +
	`		
	}
}`

func TestLookupDef(t *testing.T) {
	scriptDef := ScriptDef{}
	assert.Nil(t, scriptDef.Deserialize([]byte(scriptDefJson), nil, nil, "", nil))

	re := regexp.MustCompile(`"idx_read_batch_size": [\d]+`)
	assert.Contains(t,
		scriptDef.Deserialize([]byte(re.ReplaceAllString(scriptDefJson, `"idx_read_batch_size": 50000`)), nil, nil, "", nil).Error(),
		"cannot use idx_read_batch_size 50000, expected <= 20000")

	re = regexp.MustCompile(`"right_lookup_read_batch_size": [\d]+`)
	assert.Contains(t,
		scriptDef.Deserialize([]byte(re.ReplaceAllString(scriptDefJson, `"right_lookup_read_batch_size": 50000`)), nil, nil, "", nil).Error(),
		"cannot use right_lookup_read_batch_size 50000, expected <= 20000")

	re = regexp.MustCompile(`"filter": "[^"]+",`)
	assert.Contains(t,
		scriptDef.Deserialize([]byte(re.ReplaceAllString(scriptDefJson, `"filter": "aaa",`)), nil, nil, "", nil).Error(),
		"cannot parse lookup filter condition")
	assert.Contains(t,
		scriptDef.Deserialize([]byte(re.ReplaceAllString(scriptDefJson, `"filter": "123",`)), nil, nil, "", nil).Error(),
		"cannot evaluate lookup filter expression [123]: [expected type bool, but got int64 (123)]")
	assert.Nil(t,
		scriptDef.Deserialize([]byte(re.ReplaceAllString(scriptDefJson, ``)), nil, nil, "", nil))

	re = regexp.MustCompile(`"join_on": "[^"]+",`)
	assert.Contains(t,
		scriptDef.Deserialize([]byte(re.ReplaceAllString(scriptDefJson, `"join_on": "r.order_id,r.order_status",`)), nil, nil, "", nil).Error(),
		"lookup joins on 2 fields, while referenced index idx_order_items_order_id uses 1 fields, these lengths need to be the same")
	assert.Contains(t,
		scriptDef.Deserialize([]byte(re.ReplaceAllString(scriptDefJson, `"join_on": "",`)), nil, nil, "", nil).Error(),
		"failed to resolve lookup for node order_item_date_inner: [expected a comma-separated list of <table_name>.<field_name>, got []]")
}
