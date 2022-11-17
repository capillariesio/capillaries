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

- [lookup](../test/code/lookup/README.md): comprehensive [lookup](glossary.md#lookup) test
- [py_calc](../test/code/py_calc/README.md): focuses on [custom processor](glossary.md#table_custom_tfm_table) implementation - [py_calc](glossary.md#py_calc-processor)
- [tag_and_denormalize](../test/code/tag_and_denormalize/README.md): focuses on [custom processor](glossary.md#table_custom_tfm_table) implementation - [tag_and_denormalize](glossary.md#tag_and_denormalize-processor)

All tests require running Cassandra and RabbitMQ containers (see [Getting started](started.md) for details)
