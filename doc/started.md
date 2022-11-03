# Getting started

## Dev machine pre-requisites

1. Install Docker desktop with Docker Compose

2. If you are running Windows, make sure Ubuntu WSL is functional and Docker has "Use the WSL 2 based engine" and "Enable integration with my default WSL distro" settings on. **All commands referenced in this document should be run from WSL**. 
   
3. Install [Go](https://go.dev)

4. Run RabbitMQ Docker container using default bridge network:
```
docker pull rabbitmq:3-management 
docker run -d --hostname my-rabbit -p 15672:15672 -p 5672:5672 --ip 172.17.0.2 rabbitmq:3-management
```

More about RabbitMQ setup [here](binconfig.md#amqp) and [here](glossary.md#rabbitmq-setup).

5. Run Cassandra container using default bridge network:
```
docker pull cassandra 
docker run -d --hostname my-cassandra -p 9042:9042 --ip 172.17.0.3 cassandra
```

More about Cassandra setup [here](binconfig.md#cassandra) and [here](glossary.md#cassandra-setup).

6. You may also want to make sure you have cqlsh tool installed, it may be helpful when exploring Capillaries table structure.

## Run lookup integration test

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

If it succeeds, try it with RabbitMQ:

Run Capillaries [daemon](glossary.md#daemon):

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
- [Toolbelt and Daemon configuration](binconfig.md)
- [Script configuration](scriptconfig.md)
