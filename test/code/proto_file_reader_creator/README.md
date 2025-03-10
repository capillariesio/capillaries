# proto_file_reader_creator integration test

Created using Ubuntu WSL. Other Linux flavors and MacOS may require edits.

## Workflow

There are no Capillaries scripts for this test in the Capillaries codebase. The scripts are generated on the fly by [1_generate_scripts.sh](./1_generate_scripts.sh) and saved to `/tmp/capi_cfg/proto_file_reader_creator_quicktest`.  After the scripts are generated, you can generate the diagram using this command:

```
go run capitoolbelt.go validate_script -script_file=/tmp/capi_cfg/proto_file_reader_creator_quicktest/script_csv.json -detail=idx
```

![drawing](../../../doc/viz-proto-file-reader-creator.svg)

## What's tested:

- toolbelt `proto_file_reader_creator` command
- [file_table](../../../doc/glossary.md#file_table) read from single file (csv and parquet) for all supported data types
[table_file](../../../doc/glossary.md#table_file)table_file write to single file (csv and parquet) for all supported data types
- single-run script execution

## How to test

See [integration tests](../../../doc/testing.md#integration-tests) section for generic instructions on how to run integration tests.
