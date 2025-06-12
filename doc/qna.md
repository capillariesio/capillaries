Q&A
===

## Data connections

Q. What kind of data providers can I use as data source? What media the results can be written to?

A. At the moment, Capillaries can only read and write CSV and Parquet files. It can read from:
- file system
- http/https
- [sftp](./glossary.md#sftp-uris) URIs
- [S3 buckets](./glossary.md#s3-uris)

and write to:
- file system
- [sftp](./glossary.md#sftp-uris) URIs
- [S3 buckets](./glossary.md#s3-uris)

## Getting started with Capillaries scripts

Q. I have my data file(s) ready, but I am not sure how to create a Capillaries script that would process them. What's the best way to start?

A. You may want to try [Toolbelt](./glossary.md#toolbelt) `proto_file_reader_creator` command, it can read your source file and generate a simple Capillaries script that reads data from your file in `/tmp/capi_in`, saves it to the Capillaries table, and writes it back to the file with the same name in `/tmp/capi_out`. Check out [proto_file_reader_creator integration test](../test/code/proto_file_reader_creator/README.md), it creates Capillaries scripts on the fly and runs them against your Capillaries setup.

Make sure the script this command generated from your data file works as you expect, and add more [processsing nodes](./glossary.md#script-node) to it.

## Limits

Q. Is there a limit on the number of nodes, runs, and indexes?

A. The number of nodes in the script and runs performed for a keyspace are virtually unlimited. But keep in mind that each run-node pair creates a separate [table](glossary.md#table) in Cassandra (if an index is created, account for an extra table for each run-node-index triad), and Cassandra does not scale well when the number of tables in a keyspace exceeds a couple of hundreds. As of 2025, I would not try scripts with more than 100 nodes.

## NULL support

Q. I can't see any code/example that works with NULLs. Are they supported?

A. There is no support for NULL values. To mitigate it, Capillaries offers support for custom default values. See `default_value` in [Table write field definition](glossary.md#table-writer-field-definition). Whever an empty value found in the source CSV or Parquet file, this default_value will be written to the [table](glossary.md#table).

## Re-processing granularity

Q. I can see that Capillaries offers [runs](glossary.md#run) as a tool to handle scenarios when some data was not processed properly. After making all necessary fixes, operators can re-start a run (or multiple runs) to overwrite data for all nodes affected by this run - in intermediate Cassandra [tables](glossary.md#table) and in the result files. But it may unnecessarily affect too many nodes and take too long. Can I re-process a single failed [script node](glossary.md#script-node)? A single failed [batch](glossary.md#data-batch)?

A. Re-processing nodes: yes, to some extent. But it has to be part of the script design. You can design your Capillaries [script](glossary.md#script) in a way so the node in question and all its dependants can only be started [manually](scriptconfig.md#start_policy). This means that even on successful script execution, the operator will have to manually start a run that processes only one node in question, and manually start a run that processes its dependants.  

Re-running single batches: no. Capillaries [data tables](glossary.md#data-table) that hold results of a run are immutable, re-running batches would violate this restriction.

## dead-letter-exchange

Q. When a run is started, I can see RabbitMQ messages created immediately for every batch, and every node affected by the run. And those messages linger in the queue for a while until the node is ready for processing. Why doesn't Capillaries send RabbitMQ messages to a node only after dependency node processing is complete?

A. Because it would be extremely hard (if even possible) to guarantee the atomicity/idempotency of batch handling by code that:
- modifies [data tables](glossary.md#data-table)
- modifies [workflow tables](glossary.md#workflow-table)
- generates the next set of RabbitMQ messages

The trickiest part would be to guarantee that only one copy of a batch message for a specific node is created and handled. The approach that Capillaries takes (creating a set of messages in one shot at the run start) guarantees it. If a node is not ready to process the batch message, it goes to the the dead letter exchange where it waits for [dead_letter_ttl](binconfig.md#dead_letter_ttl) milliseconds and re-routed to the [queue](glossary.md#processor-queue) again.

[This article](https://www.cloudamqp.com/blog/when-and-how-to-use-the-rabbitmq-dead-letter-exchange.html) explains RabbitMQ dead letter exchange use.

## RabbitMQ failures

Q. Can Capilaries survive RabbitMQ service disruption?

A. No. Guaranteed delivery of RabbitMQ messages for each Capillaries [batch](glossary.md#data-batch) is one of the cornerstones of Capillaries architecture.

## Cassandra only?

Q. Can Capillaries use an internal database engine other than Cassandra?

A. No. One of the goals of Capillaries design is to use Cassandra's scalability features to their greater extent. Things like partition/clustering keys and LWTs are the cornersones of Capillaries architecture.   

## External data acquisition/processing

Q. For each row in my [data table](glossary.md#data-table), I need to acquire some new data from an external source (say, via web service), providing some row fields as arguments.

A. Start a run that dumps the table into files via [file writer](glossary.md#table_file) with some unique row identifiers, acquire data from the external source, save acquired data into new files that use the same unique row identifiers, and start a run that uses those new files.

## UI

Q. Is there a UI for Capillaries?

A. Yes. See [Capillaries UI](../ui/README.md) project, which is a simple web single-page application that shows the status of every [run](glossary.md#run) in every [keyspace](glossary.md#keyspace). UI requirements tend to be very business-specific, it's not an easy task to come up with a cookie-cutter UI framework that would be flexible enough. Dedicated solution developers are encouraged to develop their own UI for Capillaries workflows, using [Capillaries Webapi](glossary.md#webapi) as a back-end and [Capillaries UI](../ui/README.md) as an example.

Also please note that [Toolbelt](glossary.md#toolbelt) can produce visual diagrams - see [Toolbelt](glossary.md#toolbelt) `validate_script`, `get_run_status_diagram` commands. Theay are not a replacement for even a rudimentary UI, but can help visualize some things.

## Can Capillaries run in a Docker container?

Yes. See Capillaries [100% Docker-based demo](started.md#run-100-dockerized-capillaries-demo).

## RabbitMQ in the cloud?

Q. Can I run Capillaries with RabbitMQ hosted in the cloud?

A. Yes. Capillaries were successfully tested agains AWS RabbitMQ broker.

## Cassandra in the cloud?

Q. Can I run Capillaries against cloud-based Cassandra?

A. Update 2025: version 1.1.25 adds support for Amazon Keyspaces. As of 2025, this is a relatively new feature, so we have to watch for Amazon Keyspaces' pecularities. The list includes:
- everytime a keyspace or a table is created or deleted, Capillaries has to ensure the operation is completed using Amazon-specific system_schema_mcs keyspace; keyspace or table creation may take about a minute
- gocql library does not support com.amazonaws.cassandra.DefaultPartitioner, so make sure your Amazon Keyspaces configuration has org.apache.cassandra.dht.Murmur3Partitioner setting set
- Amazon Keyspaces has row size limit of 1mb
- Amazon Keyspaces has table name limit of 48 characters
- Amazon Keyspaces only supports up to 30 unconditional INSERT, UPDATE, or DELETE commands in unlogged batches
- Amazon Keyspaces supports only up to 100 values in the IN operator
- [Amazon Keyspaces pagination](https://docs.aws.amazon.com/keyspaces/latest/devguide/paginating-results.html) may return fewer records than requested, so Capillaries should watch pagestate closely
For a sample Amazon Keyspaces configuration, see [sample_amazon_keyspaces_config.env](../sample_amazon_keyspaces_config.env).

WARNING: please keep in mind that without setting read/write capacity units, Amazon Keyspaces may be extremely slow, and with those capacity units specified - extremely expensive; exercise extreme caution when setting them.

## What's next?

Q. What are the potential directions for improvement?

A. Here are some, in no particular order:

1. Performance enhancements, especially those related to the efficient use of Cassandra.

2. Read/write from/to other file formats, maybe databases. Update 2023: Apache Parquet support was added, see [Parquet reader](glossary.md#parquet-reader-column-properties) and [Parquet writer](glossary.md#parquet-specific-writer-column-properties).

3. Creating node configuration is a tedious job. Consider adding a toolbelt command that takes a CSV or Parquet file as an input and generates JSON for a corresponding file_table/table_file node. Update 2023: done, see [proto_file_reader_creator test](../test/code/proto_file_reader_creator/README.md).

4. Is the lack of NULL vsalues support a deal-breaker? Update March 2024: support for *_if aggregate functions was added, it should help mitigate the lack of NULL support.

5. Python formulas. (a) Need a strategy to mitigate potential security threats introduced by py_calc. SELinux/AppArmor? (b) Consider generic lambda functions?

6. Keep an eye on Azure/AWS/GCP progress with Cassandra-compatible databases (latency!) and RabbitMQ offerings. Update 2025: version 1.1.25 works with Amazon RabbitMQ broker and Amazon Keyspaces.

7. Select distinct field values from a table: it can be implemented easily using a set, but it will not scale and it will be limited by the size of the map. Alternatively, it can be implemented using Cassandra features, but it will require Capillaries to support tables without [rowid](glossary.md#rowid) (so the unique values are stored in a partitioning key field). Update March 2024: done, see [distinct_table](./glossary.md#distinct_table) node.

8. Keep adding support for Go library functions

9. Bugs. One of the biggest painpoints is connectivity failure handling. 