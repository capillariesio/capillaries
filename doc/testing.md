# Testing

## Unit tests and code coverage

cd into any directory under pkg/ and run 
```
go test -v
```

To see test code coverage:
```
go test -v -coverprofile=/var/tmp/capillaries.p
go tool cover -html=/var/tmp/capillaries.p -o=/var/tmp/capillaries.html
```
and open /var/tmp/capillaries.html in a web browser.

Or, from the root directory:

```
go test -coverprofile=/var/tmp/capillaries.p -cover $(go list ./pkg/... | grep -e .*\.\/cql$ -e .*\.\/custom$ -e .*\.\/eval$ -e .*\.\/sc$ -e .*\.\/wf$)
go tool cover -html=/var/tmp/capillaries.p -o=/var/tmp/capillaries.html
```

## Integration tests

There is a number of extensive integration tests that cover a big part of Capillaries script, database, and workflow functionality:

- [lookup](../test/code/lookup/README.md): comprehensive [lookup](glossary.md#lookup) test
- [py_calc](../test/code/py_calc/README.md): focuses on [custom processor](glossary.md#table_custom_tfm_table) implementation - [py_calc](glossary.md#py_calc-processor)
- [tag_and_denormalize](../test/code/tag_and_denormalize/README.md): focuses on [custom processor](glossary.md#table_custom_tfm_table) implementation - [tag_and_denormalize](glossary.md#tag_and_denormalize-processor)
- [portfolio](../test/code/portfolio/README.md): exercises [lookups](glossary.md#lookup), [py_calc](glossary.md#py_calc-processor), [tag_and_denormalize](glossary.md#tag_and_denormalize-processor)

All tests require running Cassandra and (in most cases) RabbitMQ containers (see [Getting started](started.md) for details). All tests run [Toolbelt](glossary.md#toolbelt) to send work batches to the queue and to check Capillaries workflow status.

Before running an integration test (before you build Docker containers, if you choose to do so), make sure you have copied all test configuration and data files to /tmp/capi_* directories as described in [Prepare Data Directories](./started.md#prepare-data-directories).

How to run integration tests?

### Direct node execution

Run `test_exec_nodes.sh`  - the [Toolbelt](glossary.md#toolbelt) executes test's `script.json` [nodes](glossary.md#script-node) one by one, without invoking RabbitMQ workflow. Running nodes one by one is not something you want to do in production environment, but in can be particulary convenient when troubleshooting specific script nodes.

### Using RabbitMQ workflow (single run)

Make sure the [Daemon](glossary.md#daemon) is running:
- either run `go run capidaemon.go` to start it in pkg/exe/daemon
- or start the Daemon container (`docker compose -p "test_capillaries_containers" start daemon`)

Run `test_one_run.sh` - the [Toolbelt](glossary.md#toolbelt) publishes [batch messages](glossary.md#data-batch) to RabbitMQ and the [Daemon](glossary.md#daemon) consumes them and executes all script [nodes](glossary.md#script-node) in parallel as part of a single [run](glossary.md#run).

### Using RabbitMQ workflow (two runs)

Make sure the [Daemon](glossary.md#daemon) is running:
- either run `go run capidaemon.go` to start it in pkg/exe/daemon
- or start the Daemon container (`docker compose -p "test_capillaries_containers" start daemon`)

Run `test_two_runs.sh` (if it is available for the speficic test) - the [Toolbelt](glossary.md#toolbelt) publishes [batch messages](glossary.md#data-batch) to RabbitMQ and the [Daemon](glossary.md#daemon) consumes them and executes script [nodes](glossary.md#script-node) that load data from files as part of the first [run](glossary.md#run).

After the first [run](glossary.md#run) is complete, the [Toolbelt](glossary.md#toolbelt) publishes [batch messages](glossary.md#data-batch) to RabbitMQ and the [Daemon](glossary.md#daemon) consumes them and executes script [nodes](glossary.md#script-node) that process the data as part of the second [run](glossary.md#run).

This test mimics the "operator validation" scenario.

### Using RabbitMQ workflow (single run, HTTPS inputs)

This test variation is supported only by [tag_and_denormalize](../test/code/tag_and_denormalize/README.md) integration test.

Make sure the [Daemon](glossary.md#daemon) is running:
- either run `go run capidaemon.go` to start it in pkg/exe/daemon
- or start the Daemon container (`docker compose -p "test_capillaries_containers" start daemon`)

Make sure that the daemon can connect to github.com.

Same as `test_one_run.sh`, but uses GitHub as the source of configuration and input data.

Run `test_one_run_input_https.sh` - the [Toolbelt](glossary.md#toolbelt) publishes [batch messages](glossary.md#data-batch) to RabbitMQ and the [Daemon](glossary.md#daemon) consumes them and executes all script [nodes](glossary.md#script-node) in parallel as part of a single [run](glossary.md#run).

## Webapi

Make sure the [Daemon](glossary.md#daemon) is running:
- either run `go run capidaemon.go` to start it in pkg/exe/daemon
- or start the Daemon container (`docker compose -p "test_capillaries_containers" start daemon`)

Make sure the [Webapi](glossary.md#webapi) is running:
- either run `go run capiwebapi.go` to start it in pkg/exe/webapi
- or start the Webapi container (`docker compose -p "test_capillaries_containers" start webapi`)

Navigate to `http://localhost:8080` and start a new run from the UI as described in the [Getting started: Run 100% dockerized demo](./started.md#run-100-dockerized-capillaries-demo).
