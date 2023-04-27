<img src="doc/logo.svg" alt="logo" width="100"/>

# Capillaries

Capillaries is a data processing framework that:
- takes care of the scalability issues and intermediate data store, allowing users to focus on data transforms and data quality control;
- fills the gap between distributed, scalable data processing/integration solutions and the need to produce enriched, customer-ready, production-quality, human-curated data within SLA time limits.

## Why Capillaries?
![Capillaries: before and after](doc/beforeafter.png)


|             | BEFORE | AFTER |
| ----------- | ------ |------ |
| Data aggregation | SQL joins | Capillaries [lookups](doc/glossary.md#lookup) in Cassandra + [Go expressions](doc/glossary.md#go-expressions) (scalability, parallel execution) |
| Data filtering | SQL queries, custom code | [Go expressions](doc/glossary.md#go-expressions) (scalability, maintainability) |
| Data transform | SQL expressions, custom code | [Go expressions](doc/glossary.md#go-expressions), Python [formulas](doc/glossary.md#py_calc-processor) (parallel execution, maintainability) |
| Intermediate data storage | Files, relational databases | on-the-fly-created Cassandra [keyspaces](doc/glossary.md#keyspace) and [tables](doc/glossary.md#table) (scalability, maintainability) |
| Workflow execution | Shell scripts, custom code, workflow frameworks | RabbitMQ as the Single Point of Failure + workflow status stored in Cassandra (parallel execution, fault tolerance, incremental computing) |
| Workflow monitoring and interaction | Custom solutions | Capillaries [UI](ui/README.md), [Toolbelt](doc/glossary.md#toolbelt) utility, [API](doc/api.md), [Web API](doc/glossary.md#webapi) (transparency, operator validation support) |
| Workflow management | Shell scripts, custom code | Capillaries [script file](doc/glossary.md#script) with [DAG](doc/glossary.md#dag) |

## Getting started

On Mac, WSL or Linux, run:

```
git clone https://github.com/capillariesio/capillaries.git
cd capillaries
./copy_demo_data.sh
docker-compose -p "test_capillaries_containers" up
```

Wait until all containers are started and Cassandra is fully initialized (it will log something like `Created default superuser role 'cassandra'`). Now Capillaries is ready to process data.

Navigate to `http://localhost:8080`, click "New run" and start a new data processing run with the following parameters:

| Field | Value |
|- | - |
| Keyspace | portfolio_quicktest |
| Script URI | /tmp/capi_cfg/portfolio_quicktest/script.json |
| Script parameters URI | /tmp/capi_cfg/portfolio_quicktest/script_params.json |
| Start nodes |	1_read_accounts,1_read_txns,1_read_period_holdings |

A new keyspace `portfolio_quicktest` will appear in the keyspace list. Click on it and watch the run complete - nodes `7_file_account_period_sector_perf` and `7_file_account_year_perf` should produce result files:

```
cat /tmp/capi_out/portfolio_quicktest/account_period_sector_perf.csv
cat /tmp/capi_out/portfolio_quicktest/account_year_perf.csv
```

For more details about getting started, see [Getting started](doc/started.md). For more details about this particular demo, see Capillaries blog: [Use Capillaries to calculate ARK portfolio performance](https://capillaries.io/blog/2023-04-08-portfolio/index.html)

## Capillaries in depth

### [What it is and what it is not](doc/what.md) (with a use case discussion and diagrams)
### [Getting started](doc/started.md) (how to run a quick Docker-based demo without compiling a single line of code)
### [Testing](doc/testing.md)
### [Toolbelt, Daemon, and Webapi configuration](doc/binconfig.md)
### [Script configuration](doc/scriptconfig.md)
### [Capillaries UI](ui/README.md)
### [Capillaries API](doc/api.md)
### [Capillaries deploy tool: Openstack cloud deployment](test/deploy/README.md)
### [Glossary](doc/glossary.md)
### [Q & A](doc/qna.md)
### [MIT License](LICENSE)

(C) 2023 kleines.hertz[at]protonmail.com