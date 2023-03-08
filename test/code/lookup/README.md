# Lookup integration test

Created using Ubuntu WSL. Other Linux flavors and MacOS may require edits.

## Workflow

The [DOT diagram](../../../doc/glossary.md#dot-diagrams) generated with
```
go run capitoolbelt.go validate_script -script_file=../../../test/data/cfg/lookup_quicktest/script.json -params_file=../../../test/data/cfg/lookup_quicktest/script_params_two_runs.json -idx_dag=true
```
and rendered in https://dreampuf.github.io/GraphvizOnline :

![drawing](../../../doc/dot-lookup.svg)

## What's tested:

- table_lookup_table with parallelism (10 batches), all suported types of joins (inner and left outer, grouped and not)
- file_table read from single file
- table_file with top/limit/order
- single-run (test_one_run.sh) and multi-run (test_two_runs.sh) script execution

Multi-run test simulates the scenario when an operator validates loaded order and order item data before proceeding with joining orders with order items.

## How to test

See [integration tests](../../../doc/testing.md#integration-tests) section for generic instructions on how to run integration tests.

## Possible edits

Play with number of total line items (see "-items=..." in [1_create_quicktest_data.sh](1_create_quicktest_data.sh)).
  
## References:

Data model design: Brazilian E-Commerce public dataset (https://www.kaggle.com/datasets/olistbr/brazilian-ecommerce)