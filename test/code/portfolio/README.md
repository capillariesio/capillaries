# Portfolio performance calculation integration test

Created using Ubuntu WSL. Other Linux flavors and MacOS may require edits.

## Workflow

The [DOT diagram](../../../doc/glossary.md#dot-diagrams) generated with
```
go run capitoolbelt.go validate_script -script_file=../../../test/data/cfg/portfolio_quicktest/script.json -params_file=../../../test/data/cfg/portfolio_quicktest/script_params.json -idx_dag=true
```
and rendered in https://dreampuf.github.io/GraphvizOnline :

![drawing](./doc/dot-portfolio.svg)

Or, here is a bit more low-level walkthrough

### Data preparation

First few script nodes read CSV files and produce Cassandra records with JSON:

![Prepare data](doc/prepare-data.svg)

In other words, we have to collect all holdings/txn data for an account in a single record, so it can be processed by [py_calc](../../../doc/glossary.md#py_calc-processor).

### Calculation

For each account_id, [py_calc](../../../doc/glossary.md#py_calc-processor) node calc_account_period_perf takes both JSON fields and calculates annualized returns for the range specified by `period_start_eod` and `period_end_eod` script [parameters](../../../doc/scriptconfig.
md#template-parameters).

After calculating portfolio returns, we end up with data looking as follows:

| account_id | perf_json |
| --- | --- |
| ARKK | {"2021": {"All": {"cagr": -0.2398, "twr": -0.2398}, "Communication Services": {"cagr": -0.3183, "twr": -0.3183}, "Consumer Cyclical": {"cagr": 0.1764, "twr": 0.1764}, ... } |
| ARKW | {"2021": {"All": {"cagr": -0.1949, "twr": -0.1949}, "Communication Services": {"cagr": -0.293, "twr": -0.293}, "Consumer Cyclical": {"cagr": -0.0385, "twr": -0.0385}, ... } |

### Reporting

Technicaly speaking, we already have what we want. Next few steps make this JSON data relational:

| ARK fund | Period | Sector | Time-weighted annualized return |
| --- |  --- | --- | --- |
| ARKK | 2021 | All | -23.98 |
| ARKK | 2021 |  Communication Services |  -31.83 |
| ARKK | 2021 |  Consumer Cyclical |  17.64 |
| ARKW | 2021 |  All |  -19.49 |
| ARKW | 2021 |  Communication Services |  -29.30 |
| ARKW | 2021 |  Consumer Cyclical |  -3.85 |

See results in /tmp/capi_out/portfolio_quicktest.

## What's tested:

- file_table read from file directly into JSON fields
- table_lookup_table with parallelism, left outer grouped joins, string_agg() aggregate function
- py_calc calculations taking JSON as input and producing JSON
- table_file with top/orderto produce ordered performance data matrix

## How to test

### Direct node execution

Run [test_exec_nodes.sh](test_exec_nodes.sh)  - the [Toolbelt](../../../doc/glossary.md#toolbelt) executes [script](../../data/cfg/portfolio_quicktest/script.json) [nodes](../../../doc/glossary.md#script-node) one by one, without invoking RabbitMQ workflow.

### Using RabbitMQ workflow (single run)

Make sure the [Daemon](../../../doc/glossary.md#daemon) is running:
- either run `go run capidaemon.go` to start it in pkg/exe/daemon
- or start the Daemon container (`docker compose -p "test_capillaries_containers" start daemon`)

Run [test_one_run.sh](test_one_run.sh) - the [Toolbelt](../../../doc/glossary.md#toolbelt) publishes [batch messages](../../../doc/glossary.md#data-batch) to RabbitMQ and the [Daemon](../../../doc/glossary.md#daemon) consumes them and executes all [script](../../data/cfg/portfolio_quicktest/script.json) [nodes](../../../doc/glossary.md#script-node) in parallel as part of a single [run](../../../doc/glossary.md#run).

## Webapi

Make sure the [Daemon](../../../doc/glossary.md#daemon) is running:
- either run `go run capidaemon.go` to start it in pkg/exe/daemon
- or start the Daemon container (`docker compose -p "test_capillaries_containers" start daemon`)

Make sure the [Webapi](../../../doc/glossary.md#webapi) is running:
- either run `go run capiwebapi.go` to start it in pkg/exe/webapi
- or start the Webapi container (`docker compose -p "test_capillaries_containers" start webapi`)

The test runs the same scenario as the previous [two runs test](#using-rabbitmq-workflow-two-runs) above, but uses [Webapi](../../../doc/glossary.md#webapi) instead of the [Toolbelt](../../../doc/glossary.md#toolbelt)

## Possible edits

Stretch goal: change portfolio_calc.py and script.json (period tags) to produce montly, not quarterly returns for each account.

## How accurate are these numbers?

Not very. The data was borrowed from free projects that scrape ARK websites, there are a few problems with it:
- we do not have exact trade prices, we use EOD prices instead
- holding information for some funds and stocks is missing, so we have to fill the gaps by creating non-existing trades
- exact price information would take too much space in our test price provider, so we store only some key points and interpolate price for specific stock and specific date
- we do not know how close our TWR/CAGR calculation formula is to the method used in the official calculation of ARK funds performance

Given all that, the numbers this test gets are relatively close to the official returns published by ARK, compare these two files in /mnt/capi_out/portfolio_quicktest: account_year_perf_official.csv,
account_year_perf_baseline.csv.
