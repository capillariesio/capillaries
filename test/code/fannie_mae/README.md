# Fannie Mae integration test

Created using Ubuntu WSL. Other Linux flavors and MacOS may require edits.

## fannie_mae_quicktest vs fannie_mae_bigtest

This test comes in two flavors. 

fannie_mae_quicktest has all data ready, it just has to be copied to /tmp/capi_*, and you can run the test. Root-level [copy_demo_data.sh](../../../copy_demo_data.sh) script does that, among other things.

fannie_mae_quicktest works with 60345 mortgages participating in the 2023 R08 G1 CAS risk transfer program. It uses only payment records from the Oct 20, 2023 report, so there are exactly 60345 payment records.

fannie_mae_bigtest is a variation that uses large number of payment records borrowed from capillaries-fanniemae repo (~1.6m mortgages, ~25m payment records).

## Workflow

The [DOT diagram](../../../doc/glossary.md#dot-diagrams) generated with
```
go run capitoolbelt.go validate_script -script_file=../../../test/data/cfg/fannie_mae_quicktest/script.json -params_file=../../../test/data/cfg/fannie_mae_bigtest/script_params.json -idx_dag=true
```
and rendered in https://dreampuf.github.io/GraphvizOnline :

![drawing](../../../doc/dot-fanniemae.svg)

Full transcript of what the result of each script node looks like in Cassandra - [transcript_fannie_mae.md](./transcript_fannie_mae.md).

## What's tested:

- [distinct_table](../../../doc/glossary.md#distinct_table) node type
- [file_table](../../../doc/glossary.md#file_table) read from multiple files file
- [table_file](../../../doc/glossary.md#table_file) with [top/limit/order](../../../doc/scriptconfig.md#wtop)
- [table_custom_tfm_table](../../../doc/glossary.md#table_custom_tfm_table) custom processor [py_calc](../../../doc/glossary.md#py_calc-processor) calculations taking JSON as input and producing JSON
- [table_lookup_table](../../../doc/glossary.md#table_lookup_table) with parallelism, left outer grouped joins, string_agg() aggregate function
- some *_if aggregate functions
- single-run script execution

## How to test

See [integration tests](../../../doc/testing.md#integration-tests) section for generic instructions on how to run integration tests.