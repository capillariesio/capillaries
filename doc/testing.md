# Testing

## Unit tests and code coverage

cd into a directory under pkg/ and run 
```
go test - v
```

To see test code coverage:
```
go test -v -coverprofile=/var/tmp/capillaries.p
go tool cover -html=/var/tmp/capillaries.p -o=/var/tmp/capillaries.html
```
and open /var/tmp/capillaries.html in a web browser.

## Integration tests
There is a number of extensive integration tests that cover a big part of Capillaries script, database, and workflow functionality:
- [lookup](../test/lookup/README.md): comprehensive [lookup](glossary.md#lookup) test
- [py_calc](../test/py_calc/README.md): focuses on [custom processor](glossary.md#table_custom_tfm_table) implementation - [py_calc](glossary.md#py_calc-processor)
- [tag_and_denormalize](../test/tag_and_denormalize/README.md): focuses on [custom processor](glossary.md#table_custom_tfm_table) implementation - [tag_and_denormalize](glossary.md#tag_and_denormalize-processor)

All tests require running Cassandra and RabbitMQ containers (see [Getting started](started.md) for details)

## Docker testing

[py_calc integration test](../test/py_calc/README.md) has everything ready to run against containerized [Daemon](glossary.md#daemon).

### Build image

To build Capillaries daemon image, run from the Capillaries source root directory:
```
docker build --pull --rm -f Dockerfile -t capillaries:latest .
```

### Create directory for the shared volume

The test requires some volume to be shared by the host and the container. It's important that the volume can be referenced by the same URL by the host (that runs the [Toolbelt](glossary.md#toolbelt)) and the container (that runs the [Daemon](glossary.md#daemon)). For simplicity, just create a top-level directory on the host machine (it will be mounted in the container when we run it in the next step):
```
mkdir /capillaries_src_root
```

### Start the container

At this point, you should already have Cassandra and RabbitMQ running in their containers. Get their IP addresses (for example, by running `hostname-i` in the container shell). Start Capillaries container, the example below assumes:
- RabbitMQ has address 172.17.0.2
- Cassandra has address 172.17.0.3
- image `capillaries` was successfully built (see above)
- directory /capillaries_src_root exists on the host (see above)
```
docker run -t -e AMQP_URL='amqp://guest:guest@172.17.0.2/' -e CASSANDRA_HOSTS='["172.17.0.3"]' -v /capillaries_src_root:/capillaries_src_root --name capillaries_daemon capillaries
```
Check out container output for errors, Capillaries daemon should connect to RabbitMQ and start listening.

### Run the test

From test/py_calc, run `./test_one_run_docker.sh`.

### 100% containerized tests?

The user may be tempted to run tests against dockerized Capillaries daemon without even installing Go. This is possible in theory, but some component has to analyze the script file and to post messages to RabbitMQ. This is exactly what [Toolbelt](glossary.md#toolbelt) `start_run` command does (see [API reference](api.md)). And you need Go to build the Toolbelt.
