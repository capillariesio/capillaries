# <img src="doc/logo.svg" alt="logo" width="60"/> Capillaries <div style="float:right;"> [![coveralls](https://coveralls.io/repos/github/capillariesio/capillaries/badge.svg?branch=main)](https://coveralls.io/github/capillariesio/capillaries?branch=main) [![goreport](https://goreportcard.com/badge/github.com/capillariesio/capillaries)](https://goreportcard.com/report/github.com/capillariesio/capillaries) [![Go Reference](https://pkg.go.dev/badge/github.com/capillariesio/capillaries.svg)](https://pkg.go.dev/github.com/capillariesio/capillaries)</div>


Capillaries is a data processing framework that:
- addresses scalability issues and manages intermediate data storage, enabling users to concentrate on data transforms and quality control;
- bridges the gap between distributed, scalable data processing/integration solutions and the necessity to produce enriched, customer-ready, production-quality, human-curated data within SLA time limits.

## Why Capillaries?
![Capillaries: before and after](doc/beforeafter.png)


|             | BEFORE | AFTER |
| ----------- | ------ |------ |
| Cloud-friendly | Depends | Can be deployed to the cloud within minutes; Docker-ready |
| Data aggregation | SQL joins | Capillaries [lookups](doc/glossary.md#lookup) in Cassandra + [Go expressions](doc/glossary.md#go-expressions) (scalability, parallel execution) |
| Data filtering | SQL queries, custom code | [Go expressions](doc/glossary.md#go-expressions) (scalability, maintainability) |
| Data transform | SQL expressions, custom code | [Go expressions](doc/glossary.md#go-expressions), Python [formulas](doc/glossary.md#py_calc-processor) (parallel execution, maintainability) |
| Intermediate data storage | Files, relational databases | on-the-fly-created Cassandra [keyspaces](doc/glossary.md#keyspace) and [tables](doc/glossary.md#table) (scalability, maintainability) |
| Workflow execution | Shell scripts, custom code, workflow frameworks | RabbitMQ as scheduler, workflow status stored in Cassandra (parallel execution, fault tolerance, incremental computing) |
| Workflow monitoring and interaction | Custom solutions | Capillaries [UI](ui/README.md), [Toolbelt](doc/glossary.md#toolbelt) utility, [API](doc/api.md), [Web API](doc/glossary.md#webapi) (transparency, operator validation support) |
| Workflow management | Shell scripts, custom code | Capillaries configuration: [script file](doc/glossary.md#script) with [DAG](doc/glossary.md#dag), Python [formulas](doc/glossary.md#py_calc-processor) |

## Getting started

On Mac, WSL or Linux, run in bash shell:

```
git clone https://github.com/capillariesio/capillaries.git
cd capillaries
./copy_demo_data.sh
docker-compose -p "test_capillaries_containers" up
```

Wait until all containers are started and Cassandra is fully initialized (it will log something like `Created default superuser role 'cassandra'`). Now Capillaries is ready to process sample demo input data according to the sample demo scripts (all copied by copy_demo_data.sh above).

Navigate to `http://localhost:8080` to see [Capillaries UI](./doc/glossary.md#capillaries-ui).

Start a new Capillaries [data processing run](./doc/glossary.md#run) by clicking "New run" and providing the following parameters (no tabs or spaces allowed):

| Field | Value |
|- | - |
| Keyspace | portfolio_quicktest |
| Script URI | /tmp/capi_cfg/portfolio_quicktest/script.json |
| Script parameters URI | /tmp/capi_cfg/portfolio_quicktest/script_params.json |
| Start nodes |	1_read_accounts,1_read_txns,1_read_period_holdings |

Alternatively, you can start a new [run](./doc/glossary.md#run) using Capillaries [toolbelt](./doc/glossary.md#toolbelt) by executing the following command from the Docker host machine, it should have the same effect as starting a run from the UI:

```
docker exec -it capillaries_webapi /usr/local/bin/capitoolbelt start_run -script_file=/tmp/capi_cfg/portfolio_quicktest/script.json -params_file=/tmp/capi_cfg/portfolio_quicktest/script_params.json -keyspace=portfolio_quicktest -start_nodes=1_read_accounts,1_read_txns,1_read_period_holdings
```

Watch the progress in Capillaries UI. A new keyspace `portfolio_quicktest` will appear in the keyspace list. Click on it and watch the run complete - nodes `7_file_account_period_sector_perf` and `7_file_account_year_perf` should produce result files:

```
cat /tmp/capi_out/portfolio_quicktest/account_period_sector_perf.csv
cat /tmp/capi_out/portfolio_quicktest/account_year_perf.csv
```

Log files created by Capillaries [Daemon](./doc/glossary.md#daemon), [WebAPI](./doc/glossary.md#webapi) and [UI](./doc/glossary.md#capillaries-ui) are in /tmp/capi_out.

For more details about getting started, see [Getting started](doc/started.md). For more details about this particular demo, see Capillaries blog: [Use Capillaries to calculate ARK portfolio performance](https://capillaries.io/blog/2023-04-08-portfolio/index.html). To learn how this demo runs on a bigger dataset with 14 million transactions, see [Capillaries: ARK portfolio performance calculation at scale](https://capillaries.io/blog/2023-11-15-portfolio-scale/index.html).

## Capillaries in depth

### [What it is and what it is not](doc/what.md) (use case discussion and diagrams)
### [Getting started](doc/started.md) (run a quick Docker-based demo without compiling a single line of code)
### [Testing](doc/testing.md)
### [Toolbelt, Daemon, and Webapi configuration](doc/binconfig.md)
### [Script configuration](doc/scriptconfig.md)
### [Capillaries UI](ui/README.md)
### [Capillaries API](doc/api.md)
### [Glossary](doc/glossary.md)
### [Q & A](doc/qna.md)
### [Capillaries blog](https://capillaries.io/blog/index.html)
### [MIT License](LICENSE)

(C) 2022-2024 KH (kleines.hertz[at]protonmail.com)