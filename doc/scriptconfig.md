# Script configuration

Each [processor queue](glossary.md#processor-queue) message that asks a [processor](glossary.md#processor) to handle a batch holds a reference to a [Capillaries script](glossary.md#script). To start getting familiar with script structure, see `script.json` files used in [integration tests](testing.md#integration-tests).

## Template parameters

Some settings in [script](glossary.md#script) files can be templated using curly braces like "{start_date}". Script parameters file contains a map with actual values that have to be used instead of templated ones.

Templated parameter declarations may contain data type information. For example, this script setting
```
"expected_batches_total": "{lookup_node_batches_total|number}"
```
with supplied parameter value in the parameters file
```
"lookup_node_batches_total": 10
```
will end up looking like this in the final script:
```
"expected_batches_total": 10
```
and not like this:
```
"expected_batches_total": "10"
```

Supported parameter types are "string" (default), "number", "bool".

Also, there is a small set of built-in parameters used internally, but should be paid attention to:
`{batch_idx|string}` and `{run_id|string}`. [Processors](glossary.md#processor) get values for these parameters from the execution context (execution context always has some specific run id and batch id) and write them as `fmt.Sprintf("%05d", runId)` and `fmt.Sprintf("%05d", batchIdx)`. As of this writing, this functionality is present in the [file writer](glossary.md#table_file) and covers the scenario when the user wants run id or batch id to be present in the result file name. For example, [py_calc integration test](../test/py_calc/README.md) script uses `batchIdx`.

## nodes
[Script node](glossary.md#script-node) map, in no particular order

### type
Node [processor type](glossary.md#processor-types)

### start_policy

- `auto`: Capillaries automatically start this node processing when all dependency nodes are successfully completed
- `manual`: Capillaries will start this node processing only if this node is explicitly specified on the [run](glossary.md#run) start; manual nodes are marked on the [dot diagram](dot-tag-and-denormalize.svg) with a thicker border

Mark nodes as `manual` when you want the operator to review the results of the previous [runs](glossary.md#run) before moving ahead with the rest of the script.

Default: auto.

### dependency_policy

Name of the [dependency policy](#dependency_policies) used when Capillaries decides against executing this node or waiting for some dependencies

Default: dependency policy marked as [is_default](#is_default)

### desc
[Node](glossary.md#script-node) description

### rerun_policy
- rerun: let the daemon (same instance or another) execute this batch again (default)
- fail: give up and mark this node as failed

With policy set to "rerun", batch re-run happens automatically when the binary handling the message loses connection to the message broker (RabbitMQ) after a message is consumed, but before it is acknowledged. In such a case, the message broker re-routes the message again, and it ends up being consumed by another (or by the same) message handler binary. In this scenario, the handler that handles the re-routed message needs to make sure that there are no leftovers of the first message handler activity in [data tables](glossary.md#data-table) and [index tables](glossary.md#index-table).

The following part discusses re-runs in detail and requires a good understanding of Capillaries data/index table structure and Cassandra data modeling principles.

Batch-based cleanup requires introducing batch_idx (non-key) field to [data tables](glossary.md#data-table). Before processing the message, the second handler walks through the whole [data table](glossary.md#data-table), harvests all records for the current batch_idx, and deletes data records by their [rowids](glossary.md#rowid).

Please note that **this is a slow process**, but it gives reliable results.

For [index tables](glossary.md#index-table), the second handler does not perform this clean-up, and this is why. Consider a scenario when the first handler adds a data record with unique rowid=123 and then crashes in the process. The batch message is re-routed to another instance of the handler that notes that the batch processing was started, but not finished. So, the second handler runs the cleanup for all records with this batch_idx and writes data and index records again now with different [rowids](glossary.md#rowid). 

In the [data table](glossary.md#data-table), we now have:

| rowid | explanation |
|-----|-----------|
| 456 | inserted by the second handler |
| | no matching record with rowid=123, it was removed by the cleanup procedure |

In the index table, we now have:

| key | rowid | explanation|
|---|-----|-----------|
| 'aaa' | 123 | orphan record, inserted by first handler, gracefully ignored by the second handler |
| 'aaa' | 456 | to be inserted by the second handler, the rowid potentially (random number generator collision), but highly unlikely can be 123 again |

This data example is possible for the **non-unique** idx scenario as rowid is a clustered key, and the 'aaa' 456 record will end up in the [index table](glossary.md#index-table). We make sure that our lookup implementation handles gracefully this scenario by ignoring the index record with [rowid](glossary.md#rowid) that does not have a [rowid](glossary.md#rowid) counterpart in the [data table](glossary.md#data-table).

For the **unique** index scenario ([rowid](glossary.md#rowid) is not a clustered key, so the key field must be unique), the second handler would throw an error when trying to insert the second index record. There is no way we can distinguish between this scenario (which is a valid case if a re-run happened) and the duplicate key error scenario (in which we should stop processing and complain about a duplicate key). But, since key fields are unique in this scenario, Capillaries have the luxury of cleaning up batch leftovers in the [index table](glossary.md#index-table) by key value, not by [rowid](glossary.md#rowid). So, the second handler simply deletes all index records with key 'aaa' during the cleanup, without paying attention to [rowid](glossary.md#rowid).


### r - reader
Configures table or file reader, depending on the [processor type](glossary.md#processor-types)

#### r.table
Table reader only. Name of the [data table](glossary.md#table) to read from.

#### r.expected_batches_total
Table reader only. Number of data batches to supply to the node in parallel. Choose these settings according to your hardware environment specifics.

Default: 1 (no [parallelism](glossary.md#parallellism)).

#### r.rowset_size
Table reader only. The number of data rows to read from the source table at once when processing the batch.

Default: 1000

#### r.urls
File reader only. List of files to read from. One file - one batch.

#### r.columns
File reader only. Array of file reader [column definitions](glossary.md#file-reader-column-definition)



### w - writer

Configures table or file writer, depending on the [processor type](glossary.md#processor-types)

#### w.name

Table writer only: target table name.

#### w.fields

Table writer only: map of table writer [field definition](glossary.md#table-writer-field-definition)

#### w.top

File writer only: used only when file output has to be sorted.

`order`: [order expression](glossary.md#order-expression) to be used for sorting

`limit`: maximum number of sorted rows to write; default (and maximum allowed): 500000

#### w.columns

File writer only: array of file writer [column definitions](glossary.md#file-writer-column-definition)

#### w.having

[Go expression](#glossary.mdgo-expression) used as a filter before the row/line is about to be written to the target table/file. Allows writer (`w.*`) fields (for table writer) and columns (for file writers) only.

#### w.indexes

Table writer only. index_name->[index_definition](glossary.md#index-definition) map.


## dependency_policies

Map of dependency_policy definitions. Currently, there is only one dependency policy offered: "current_active_first_stopped_nogo".

Every time Capillaries receives a queue message that tells it to handle a [script](glossary.md#script) [node](glossary.md#script-node), it checks if all dependency nodes are successfully completed. Since multiple [runs](glossary.md#run) can be involved, the decision-making process may be not trivial. This is how it works.

[DependencyPolicyChecker](../pkg/wf/dependency_policy_checker.go) looks into run history and node status history [tables](glossary.md#table) and comes up with a list of [DependencyNodeEvent](../pkg/wfdb/dependency_node_event.go) objects that gives the full history of all dependency nodes across all runs. 

[DependencyPolicyChecker](../pkg/wf/dependency_policy_checker.go) walks through the list of [DependencyNodeEvent](../pkg/wfdb/dependency_node_event.go) and applies [rules](#rules) to each event. When a [rule](#rules) is satisfied, [DependencyPolicyChecker](../pkg/wf/dependency_policy_checker.go) finishes its work and produces a command that tells Capillaries either to wait for dependencies a bit more, or to proceed with handling the node, or give up handling this node as some dependencies have failed.

### event_priority_order

[Order expression](glossary.md#order-expression) used to arrange [DependencyNodeEvent](../pkg/wfdb/dependency_node_event.go) structures before checking [rules](#rules) against them.

### rules

List of dependency rules. Each rule is a tuple of `cmd` and `expression`

`cmd`: the command produced by [DependencyPolicyChecker](../pkg/wf/dependency_policy_checker.go) when this rule is satisfied; allowed values are

- `go` - "all dependencies are ready, we can run this node"

- `wait` - "still waiting for some dependencies to complete", 

- `nogo` - "some of the dependencies failed and this node cannot be handled".

`expression`: Go expression that is evaluated for a specific [DependencyNodeEvent](../pkg/wfdb/dependency_node_event.go) (`e.*`) and returns true or false

### is_default
Dependency policy to be used when the node does not have [dependency_policy](#dependency_policy) setting set. Can be omitted if there is only one dependency policy is defined.
