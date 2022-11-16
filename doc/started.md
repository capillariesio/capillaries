# Getting started

## Docker desktop and bridge network

1. Install Docker desktop with Docker Compose.
2. Create a new bridge network `capinet`
```
docker network create --driver=bridge --subnet=10.5.0.0/16 --ip-range=10.5.0.0/24 --gateway=10.5.0.1 capinet
```
Dockerized Capillaries components discussed below make assumptions about IP addresses. If you want to make changes to the suggested network and addresses, see [docker-compose.yml](../docker-compose.yml) for details.

## RabbitMQ and Cassandra

Capillaries needs both, so you have to provide access to them. The simplest way is to run them as containers, with assigned `capinet` IP addresses.

Start RabbitMQ container:

```
docker pull rabbitmq:3-management 
docker run -d --hostname my-rabbit -p 15672:15672 -p 5672:5672 --network=capinet --ip 10.5.0.2 rabbitmq:3-management
```

More about RabbitMQ setup [here](binconfig.md#amqp) and [here](glossary.md#rabbitmq-setup).

Start Cassandra container:
```
docker pull cassandra 
docker run -d --hostname my-cassandra -p 9042:9042 --network=capinet --ip 10.5.0.3 cassandra
```

More about Cassandra setup [here](binconfig.md#cassandra) and [here](glossary.md#cassandra-setup).

## Data directories

In production environments, Capillaries server components (daemon, toolbelt, webapi) need access to configuration files, source data files and target directories. In dev environments, we want Capillaries components to access those files and directories in the uniform way: for dockerized component and for the developer-run scenarios. We use /tmp/capitest_* directories that can be acessed using the same path - from the host machine and from containers (see [docker-compose.yml](../docker-compose.yml) for volume definitions). 

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
docker compose -p "test_capillaries_containers" up -d
```

This command will create and start the following containers:
- [Daemon](glossary.md#daemon) container (performs actual data transformations)
- [Webapi](glossary.md#webapi) container (backend for Capillaries-UI) 
- [Capillaries-UI](glossary.md#capillaries-ui) container (user interface to Capillaries)

After all containers are started, you can navigate to `http://localhost:8080`. On the displayed `Keyspaces` page, click `New run` enter the following pramaters and click `OK`:

| | |
|- | - |
| Keyspace | test_lookup |
| Script URI | /tmp/capitest_cfg/lookup/script.json |
| Script parameters URI | /tmp/capitest_cfg/lookup/script_params_one_run.json |
| Start nodes | read_orders,read_order_items |

A keyspace named `test_lookup` will appear on the list, click on it and watch the started [run](glossary.md#run) handling [script nodes](glossary.md#script-node). Drop the keyspace after experimenting with it. 

You have just performed the steps that `lookup` integration step does, but you operated on the UI level, instead of calling the [Toolbelt](glossary.md#toolbelt), as integration tests do.

Check out other [integration tests](testing.md#integration-tests), look into `code` scripts and try mimicking integration test behavior from Capillaries-UI.

## Setting up your dev machine

### Windows users

If you are running Windows, you will be using WSL for development:
- make sure Ubuntu WSL is functional and Docker has "Use the WSL 2 based engine" and "Enable integration with my default WSL distro" settings on; **All commands referenced in this document should be run from WSL**
- make sure your dev environment can run and debug from WSL (for example, if you use VSCode, make sure it runs from WSL, has Remote Development pack installed and has "WSL" on the left end of the status bar).

Is there a way to develop and debug Capillaries server components in a dev environment like VSCode without running it from WSL? Yes, but you will have to solve two problems.

1. Data directories /tmp/capitest_* will not be available from Windows, so you will have to tweak all configuration files and shell scripts to reference Windows paths. This is doable, but it's a tedious job.

2. When [tag_and_denormalize] integration test runs in WSL and uses Webapi executed by Windows dev environment, `curl` command will not be able to connect to Webapi's `http://localhost:6543` because of the known WSL limitation descussed at https://github.com/microsoft/WSL/issues/5211 and at https://superuser.com/questions/1679757/how-to-access-windows-localhost-from-wsl2 . You will need to use host IP address or use `$(localhost).local` instead of `localhost` in the shell script.

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
