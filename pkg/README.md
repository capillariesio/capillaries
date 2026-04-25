# What's in the pkg

## api
High-level calls used by [Toolbelt](../doc/glossary.md#toolbelt) and [Webapi](../doc/glossary.md#webapi)

## capigraph
A library that draws node script diagrams for UI

## capimq
Simple in-house message queue that can be used as an alternative to RabbitMQ and ActiveMQ

## cql
CQL query builder. Focused purely on the language side, all connectivity functionality in pkg/db.

## ctx
Processing context used across all levels of message processing

## custom
Custom [processors](../doc/glossary.md#processor):
- [py_calc](../doc/glossary.md#py_calc-processor)
- [tag_and_denormalize](../doc/glossary.md#tag_and_denormalize-processor)

## db
Cassandra-specific

## dpc
[Dependency policy checker](../doc/scriptconfig.md#dependency_policies)

## env
Environment configuration for Capillaries [binaries](../doc/binconfig.md)

## eval
Go expression evaluation engine

## evalcapi
Capillaries-specific eval logic: types, functions

## exe
Capillaries [binaries](../doc/binconfig.md)

## gocqlmem
In-memory implementation of gocql, used for testing only

## gocqlshims
Interfaces shared by gocql and gocqlmem (ideally, should have been declared in gocql)

## l
Logging

## mq
Message Queue client for Daemon, can work with AMQP 1.0 MQ or with CapiMQ

## proc
Core [script node](../doc/glossary.md#script-node) [processor](../doc/glossary.md#processor)

## sc
Capillaries [Script configuration](../doc/scriptconfig.md) objects

## storage
Working with external media (files, maybe databases in the future)

## wfdb
Workflow db access functions

## wfmodel
Workflow model

## xfer
SSH/HTTP/SFTP utility functions used by [processors](../doc/glossary.md#processor)