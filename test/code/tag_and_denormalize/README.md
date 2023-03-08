# tag_and_denormalize integration test

Created using Ubuntu WSL. Other Linux flavors and MacOS may require edits.

## Workflow

The [DOT diagram](../../../doc/glossary.md#dot-diagrams) generated with
```
go run capitoolbelt.go validate_script -script_file=../../../test/data/cfg/tag_and_denormalize_quicktest/script.json -params_file=../../../test/data/cfg/tag_and_denormalize/script_params_two_runs.json -idx_dag=true
```
and rendered in https://dreampuf.github.io/GraphvizOnline :

![drawing](../../../doc/dot-tag-and-denormalize.svg)

## What's tested:

- file_table read from single file
- tag_and_denormalize custom processor: denormalizes products table by checking tag criteria and producing a new data row for each matching tag
- table_lookup_table with parallelism (10 batches), left outer join with grouping
- table_file with top/limit/order
- single-run (test_one_run.sh) and multi-run (test_two_runs.sh) script execution

Multi-run test simulates the scenario when an operator validates tagged products (see /data/out/tagged_products_for_operator_review.csv) before proceeding with calculating totals.

## How to test

See [integration tests](../../../doc/testing.md#integration-tests) section for generic instructions on how to run integration tests.

## References:

Data model design: Flipkart products public dataset (https://www.kaggle.com/datasets/atharvjairath/flipkart-ecommerce-dataset)