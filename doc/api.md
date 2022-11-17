# API

This section is under active development, see pkg/api directory.

The goal of Capillaries API is to allow system integrators to create solutions that can start [runs](glossary.md#run) and watch their execution progress. As of this writing, all API calls can be executed either via the [Toolbelt](glossary.md#toolbelt) command or via [Capillaries UI](glossary.md#capillaries-ui) using [Webapi](glossary.md#webapi). Some examples are below.

Drop [keyspace](glossary.md#keyspace):

```
go run toolbelt.go drop_keyspace -keyspace=<keyspace_name>
```

Get workflow status from [workflow tables](glossary.md#workflow-table):

```
go run toolbelt.go get_run_history -keyspace=<keyspace_name>
go run toolbelt.go get_node_history -keyspace=<keyspace_name> -run_ids=<comma_separated_list_of_run_ids>
go run toolbelt.go get_batch_history -keyspace=<keyspace_name> -run_ids=<comma_separated_list_of_run_ids> -nodes=<comma_separated_list_of_node_names>
```

Initiate/terminate workflow - start/stop a [run](glossary.md#run):
```
go run toolbelt.go start_run -script_file=<script_file> -params_file=<script_params_file> -keyspace=<keyspace_name> -start_nodes=<comma_separated_list_of_nodes_to start>

go run toolbelt.go stop_run -keyspace=<keyspace_name> -run_id=<run_id>

```

Most of these commands are used in [integration tests](testing.md#integration-tests).
