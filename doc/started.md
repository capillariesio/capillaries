# Getting started

## Docker desktop

Install Docker desktop with Docker Compose.

## Windows users: WSL

If you are running Windows, you will be using WSL for development, so make sure Ubuntu WSL is functional and Docker has "Use the WSL 2 based engine" and "Enable integration with my default WSL distro" settings on; **All commands referenced in this document should be run from WSL**

## Prepare data directories

In production environments, Capillaries server components ([Daemon](glossary.md#daemon), [Toolbelt](glossary.md#toolbelt), [Webapi](glossary.md#webapi)) need access to configuration files, source data files and target directories. In dev environments, we want Capillaries components to access those files and directories in the uniform way: for dockerized component and for the scenarios when [Daemon](glossary.md#daemon), [Toolbelt](glossary.md#toolbelt), and [Webapi](glossary.md#webapi) are run by developers. We use `/tmp/capitest_*` directories that can be accessed using the same path - from the host machine and from containers (see [docker-compose.yml](../docker-compose.yml) for volume definitions). 

Run these commands from the root project directory, they will create those data directories and populate them with sample configurations and sample source data:

```
mkdir /tmp/capitest_cfg
mkdir /tmp/capitest_in
mkdir /tmp/capitest_out

cp -r ./test/data/cfg/* /tmp/capitest_cfg
cp -r ./test/data/in/* /tmp/capitest_in
cp -r ./test/data/out/* /tmp/capitest_out
```

## Run 100% dockerized Capillaries demo

No coding or compiling required, just run from the root project directory:

```
docker compose -p "test_capillaries_containers" up
```

This command will create bridge network `capinet`, and will create and start the following containers:
- RabbitMQ (more about RabbitMQ setup [here](binconfig.md#amqp) and [here](glossary.md#rabbitmq-setup))
- Cassandra (more about Cassandra setup [here](binconfig.md#cassandra) and [here](glossary.md#cassandra-setup))
- [Daemon](glossary.md#daemon) container (performs actual data transformations)
- [Webapi](glossary.md#webapi) container (backend for Capillaries-UI) 
- [Capillaries-UI](glossary.md#capillaries-ui) container (user interface to Capillaries)
- Graylog container (and a couple of dependencies - MongoDB and ElasticSearch containers)

While the containers are being built and started (Cassandra will take a while to initialize, you may want to wait for `Created default superuser role 'cassandra'`), check out the source data for this demo:
```
head -10 /tmp/capitest_in/tag_and_denormalize/flipcart_products.tsv
cat /tmp/capitest_in/tag_and_denormalize/tags.csv
```

The demo will process this data as described in the [sample use scenario](what.md#sample-use).

Wait until all containers are started.

You may want to see all log output from all Capillaries components running in the containers. To do that:
- navigate to Graylog UI at `http://localhost:9000` using admin/admin credentials;
- add a new `GELF UDP` input bound to `10.5.0.13` listening on 12201, call it, say, `gelf_udp`.

Now you can navigate to Capillaries UI at `http://localhost:8080`. On the displayed `Keyspaces` page, click `New run` enter the following pramaters and click `OK`:

| Field | Value |
|- | - |
| Keyspace | test_tag_and_denormalize |
| Script URI | /tmp/capitest_cfg/tag_and_denormalize/script.json |
| Script parameters URI | /tmp/capitest_cfg/tag_and_denormalize/script_params_two_runs.json |
| Start nodes |	read_tags,read_products |

A [keyspace](glossary.md#keyspace) named `test_tag_and_denormalize` will appear on the list, click on it and watch the started [run](glossary.md#run) handling [script nodes](glossary.md#script-node).

When the run is complete, check out data processing intermediate results:
```
head -10 /tmp/capitest_out/tag_and_denormalize/tagged_products_for_operator_review.csv
```

Let's assume the operator is satisfied with those tagging results, now it's time to start the second (and final) run. Either from the root `Keyspaces` screen, or from the `test_tag_and_denormalize` matrix screen, start a new run - provide almost the same input, but `Start nodes` will look different now - this second run will start from handling `tag_totals` node and calculating totals for each tag:

| Field | Value |
|- | - |
| Keyspace | test_tag_and_denormalize |
| Script URI | /tmp/capitest_cfg/tag_and_denormalize/script.json |
| Script parameters URI | /tmp/capitest_cfg/tag_and_denormalize/script_params_two_runs.json |
| Start nodes |	**tag_totals** |

When this run is complete, see final results at:
```
cat /tmp/capitest_out/tag_and_denormalize/tag_totals.tsv
```

To see log messages in Graylog, navigate to Graylog UI again and:
- add new extractor (say, `capi_json_extractor`) to `gelf_udp` input: it will parse JSON received in the `message` field of the log message;
- add a new stream (say, `capi_all`), add a new rule to it - it should take messages from `gelf_udp`, and  select `always match` as a rule so all messages make it to this stream;
- start this new stream `capi_all` and start another run in Capillaries UI or run an integration test - you should see parsed log events in `capi_all`.

Drop the keyspace using Capillaries UI `Drop` button after experimenting with it.

You have just performed the steps that [test_tag_and_denormalize](../test/code/tag_and_denormalize/README.md) integration test does, but you operated on the UI level, instead of calling the [Toolbelt](glossary.md#toolbelt), like integration tests do.

As a next step, you can check out other [integration tests](testing.md#integration-tests), look into `code` scripts and try mimicking integration test behavior from Capillaries-UI.

## Setting up your dev machine

### Windows users

If you are running Windows, you will be using WSL for development, so make sure your dev environment can run and debug from WSL (for example, if you use VSCode, make sure it runs from WSL, has Remote Development pack installed and has "WSL" on the left end of the status bar).

Is there a way to develop and debug Capillaries server components in a dev environment like VSCode without running it from WSL? Yes, but you will have to solve two problems.

1. Data directories `/tmp/capitest_*` will not be available from Windows, so you will have to tweak all configuration files and shell scripts so they reference Windows paths. This is doable, but it's a tedious job.

2. When [test_tag_and_denormalize](../test/code/tag_and_denormalize/README.md) integration test runs in WSL and uses Webapi executed from Windows dev environment, `curl` command will not be able to connect to Webapi's `http://localhost:6543` because of the known WSL limitation descussed at https://github.com/microsoft/WSL/issues/5211 and at https://superuser.com/questions/1679757/how-to-access-windows-localhost-from-wsl2 . You will need to use host IP address or use `$(localhost).local` instead of `localhost` in the shell script.

### Go development    

Install [Go](https://go.dev) to develop, debug and run Capillaries server components - [Daemon](glossary.md#daemon), [Toolbelt](glossary.md#toolbelt), and [Webapi](glossary.md#webapi), and to run [integration tests](testing.md#integration-tests).

### UI development

Install [Node.js and npm](https://docs.npmjs.com/) to develop, debug and run [Capillaries-UI](glossary.md#capillaries-ui).

### cqlsh

You may also want to make sure you have cqlsh tool installed, it may be helpful when exploring Capillaries table structure.

## Getting familiar with integration tests

This section may help you get started with [lookup integration test](../test/code/lookup/README.md).

All settings in pkg/exe/daemon/env_config.json and pkg/exe/toolbelt/env_config.json use default RabbitMQ and Cassandra settings. If you have changed your Cassandra or RabbitMQ setup, modify both env_config.json files accordingly. More about database and queue settings:
- Cassandra [settings](binconfig.md#cassandra), general Cassandra setup [considerations](glossary.md#cassandra-setup)
- RabbitMQ [settings](binconfig.md#amqp), general RabbitMQ setup [considerations](glossary.md#rabbitmq-setup)

### 1. Direct node execution

Start with the test that executes [script nodes](glossary.md#script-node) directly, without involving RabbitMQ or Capillaries [Daemon](glossary.md#daemon):

```
cd test/code/lookup
./test_exec_nodes.sh
```

### 2. Add queue to the mix

Run Capillaries [Daemon](glossary.md#daemon) (make sure that Daemon container is not running)):

```
cd pkg/exe/daemon
go run daemon.go
```

Check out its stdout- make sure it successfully connected to RabbitMQ.

In another command line session, run the test:

```
cd test/code/lookup
./test_two_runs.sh
```

## Next steps

After getting familiar with the lookup integration test, feel free to play with other [integration tests](testing.md#integration-tests).

When you feel you are ready to tweak integration tests or create your own [script](glossary.md#script), start with reading:
- [Toolbelt, Daemon and Webapi configuration](binconfig.md)
- [Script configuration](scriptconfig.md)
