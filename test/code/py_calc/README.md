# py_calc integration test

Created using Ubuntu WSL. Other Linux flavors and MacOS may require edits.

[Daemon/Toolbelt environment configuration file](../../../doc/binconfig.md#toolbelt-daemon-and-webapi-configuration) `capi*.json` `custom_processors.py_calc.python_interpreter_path` setting assumes 'python' symbolic link points to Python3 interpreter (min Python 3.10)

User-supplied formulas are expected to reside in test/data/cfg/py_calc/py directory.

## Workflow

The [DOT diagram](../../../doc/glossary.md#dot-diagrams) generated with
```
go run capitoolbelt.go validate_script -script_file=../../../test/data/cfg/py_calc/script.json -params_file=../../../test/data/cfg/py_calc/script_params.json -idx_dag=true
```
and rendered in https://dreampuf.github.io/GraphvizOnline :

![drawing](../../../doc/dot-pycalc.svg)

## What's tested:

- table_custom_tfm_table custom processor (py_calc) with writer using values from both reader (for example, r.shipping_limit_date) and custom processor (for example, p.taxed_value); please note: p.* datatype, like decimal2 of p.taxed_value, is used by writer only, do not expect this datatype when using this field in your Python code
- file_table reading from multiple files
- table_file with top/limit/order
- table_file using file-per-batch configuration (see {batch_idx} parameter)
- table_table processor that, using Capillaries Go funtions and arithmetic operations, implements a subset (no weekday math) of calculations provided by Python processor 

## How to test

### Direct node execution

Run [test_exec_nodes.sh](test_exec_nodes.sh) - the [Toolbelt](../../../doc/glossary.md#toolbelt) executes [script](../../data/cfg/py_calc/script.json) [nodes](../../../doc/glossary.md#script-node) one by one, without invoking RabbitMQ workflow.

### Using RabbitMQ workflow

Make sure the [Daemon](../../../doc/glossary.md#daemon) is running:
- either run `go run capidaemon.go` to start it in pkg/exe/daemon
- or start the Daemon container (`docker compose -p "test_capillaries_containers" start daemon`)

Run [test_one_run.sh](test_one_run.sh) - the [Toolbelt](../../../doc/glossary.md#toolbelt) publishes [batch messages](../../../doc/glossary.md#data-batch) to RabbitMQ and the [Daemon](../../../doc/glossary.md#daemon) consumes them and executes all [script](../../data/cfg/py_calc/script.json) [nodes](../../../doc/glossary.md#script-node) in parallel as part of a single [run](../../../doc/glossary.md#run).

## Possible edits

- number of total line items (see "-items=..." in [1_create_test_data.sh](1_create_test_data.sh))
- number of input files (default is 5, see "split -d -nl/5..." in [1_create_test_data.sh](1_create_test_data.sh))

## User-supplied formulas

There are two files in `test/data/cfg/py_calc/py` directory: one contains Python functions called by Capillaries [py_calc processor](../../../doc/glossary.md#py_calc-processor), another file is a user-provided set of tests for those functions (yes, user-provided code can/should be tested too). 

## References:

Data model design: Brazilian E-Commerce public dataset (https://www.kaggle.com/datasets/olistbr/brazilian-ecommerce)