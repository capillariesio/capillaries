# What's in the pkg

## api
High-level calls used by [Toolbelt](../doc/glossary.md#toolbelt) and [Webapi](../doc/glossary.md#webapi)

## cql
CQL query builder. Focused purely on the language side, all connectivity functionality in pkg/db.

## ctx
Processing context used across all levels of RabbitMQ message processing

## custom
Custom [processors](../doc/glossary.md#processor):
- [py_calc](../doc/glossary.md#py_calc-processor)
- [tag_and_denormalize](../doc/glossary.md#tag_and_denormalize-processor)

## db
Cassandra-specific

## deploy
Utility functions used by [Deploy tool](../doc/glossary.md#deploy-tool)

## env
Environment configuration for Capillaries [binaries](../doc/binconfig.md)

## eval
Go expression evaluation engine

## exe
Capillaries [binaries](../doc/binconfig.md)

## l
Logging

## proc
Core [script node](../doc/glossary.md#script-node) [processor](../doc/glossary.md#processor)

## sc
Capillaries [Script configuration](../doc/scriptconfig.md) objects

## storage
Working with external media (files, maybe databases in the future)

## wf
RabbitMQ messages handled here

## wfdb
Workflow db access functions

## wfmodel
Workflow model

## xfer
SSH/HTTP/SFTP utility functions used by [processors](../doc/glossary.md#processor) and [Deploy tool](../doc/glossary.md#deploy-tool)