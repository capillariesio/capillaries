Q&A
===

## Limits

Q. Is there a limit on the number of nodes, runs, and indexes?

A. The number of nodes in the script and runs performed for a keyspace are virtually unlimited. But keep in mind that each run-node pair creates a separate table in Cassandra (if an index is created, account for an extra table for each run-node-index triad), and Cassandra does not scale well when the number of tables in a keyspace exceeds a couple of hundreds.

## NULL support

Q. I can't see any code/example that works with NULLs. Are they supported?

A. There is no support for NULL values. To mitigate it, Capillaries offers support for custom default values. See `default_value` in [File reader column definition](glossary.md#file-reader-column-definition) and [Table reader column definition](glossary.md#table-reader-column-definition).

## dead-letter-exchange

Q. When a run is started, I can see RabbitMQ messages created immediately for every batch, and every node affected by the run. And those messages linger in the queue for a while until the node is ready for processing. Why doesn't Capillaries send RabbitMQ messages to a node only after dependency node processing is complete?

A. Because it would be extremely hard (if even possible) to guarantee the atomicity/idempotency of batch handling by code that:
- modifies [data tables](glossary.md#data-table)
- modifies [workflow tables](glossary.md#workflow-table)
- generates the next set of RabbitMQ messages

The trickiest part would be to guarantee that only one copy of a batch message for a specific node is created and handled. The approach that Capillaries takes (creating a set of messages in one shot at the run start) guarantees it. If a node is not ready to process the batch message, it goes to the the dead letter exchange where it waits for [dead_letter_ttl](binconfig.md#dead_letter_ttl) milliseconds and re-routed to the [queue](glossary.md#processor-queue) again.

[This article](https://www.cloudamqp.com/blog/when-and-how-to-use-the-rabbitmq-dead-letter-exchange.html) explains RabbitMQ dead letter exchange use.

## External data acquisition

Q. For each row in my [data table](glossary.md#data-table), I need to acquire data from an external source (say, via web service), providing some row fields as arguments.

A. Start a run that dumps the table into files via [file writer](glossary.md#table_file) with some unique row identifiers, acquire data, save acquired data into new files that use the same unique row identifiers, and start a run that uses those new files.

## UI

Q. Is there a UI for Capillaries?

A. As of this writing (2022), no. The [Toolbelt](glossary.md#toolbelt):
- can [start/stop](api.md) [runs](glossary.md#run)
- gives very basic access to the [workflow tables](glossary.md#workflow-table)
- can produce rudimentary visuals using [DOT diagram language](glossary.md#dot-diagrams) (see `validate_script`, `get_run_status_diagram` commands)
  
but that's it. UI requirements tend to be very business-specific, it's not an easy task to come up with a cookie-cutter UI framework that would be flexible enough. Solution developers are encouraged to develop their own UI for Capillaries workflows using Capillaries [API](api.md).

## Cassandra in the cloud?

Q. Can I run Capillaries against cloud-based Cassandra?

A. While processing [nodes](glossary.md#script-node) that create [tables](glossary.md#table), Capillaries creates [keyspaces](glossary.md#keyspace) and [tables](glossary.md#table) on-the-fly as a [processor](glossary.md#processor) handles the node. As of this writing (2022), Azure CosmosDB and AWS Keyspaces have notoriously high latency. For example, Azure can complete "CREATE TABLE" command successfully, but an "INSERT" command executed immediately after that may return an error saying that the table does not exist.

This situation can be potentially mitigated by creating all tables for a specific [run](glossary.md#run) in advance. A [toolbelt](glossary.md#toolbelt) command producing CQL statements that creates all tables for a [run](glossary.md#run) may look like this:

``` 
go run toolbelt.go get_table_cql -script_file=... -params_file=... -keyspace=... -run_id=... -start_nodes=...
```

The tricky part is to specify the correct run id for a run that has not started yet.

Another tricky part is to run this CQL against the cloud infrastructure and wait until all tables are guaranteed to be created.

Bottom line: Capillaries' use of cloud-based Cassandra is questionable at the moment.

## What's next?

Q. What are the potential directions to improve Capillaries?

A. Here are some:

1. Database connectors, in addition to file read/write capabilities.

2. Creating node configuration is a tedious job. Consider adding a toolbelt command that takes a CSV file as an input and generates JSON for a corresponding file_table/table_file node.

3. Is the lack of NULL support a deal-breaker?

4. Need a strategy to mitigate potential security threats introduced by py_calc. SELinux/AppArmor?

5. Keep an eye on Azure/AWS/GCP progress with Cassandra-compatible databases (latency!) and RabbitMQ offerings.
   
6. Something generic enough and useful at the same time to:
    - build UI for operators who monitor Capillaries running user scripts
    - allow integrated solutions to control Capillaries script execution
  
7. Select distinct field values from a table: it can be implemented easily using a set, but it will not scale and it will be limited by the size of the map. Alternatively, it can be implemented using Cassandra features, but it will require Capillaries to support tables without [rowid](glossary.md#rowid) (so the unique values are stored in a partitioning key field).