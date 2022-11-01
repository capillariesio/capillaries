#!/bin/bash

keyspace="test_lookup"
scriptDir="../../../test/lookup"
outDir="../../../test/lookup/data/out"
scriptFile=$scriptDir/script.json
paramsFile=$scriptDir/script_params_one_run.json

SECONDS=0

[ ! -d "./data/out" ] && mkdir ./data/out
pushd ../../pkg/exe/toolbelt
set -x
go run toolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=read_orders
go run toolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=read_order_items

go run toolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=order_item_date_inner
go run toolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=order_item_date_left_outer
go run toolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=order_date_value_grouped_inner
go run toolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=order_date_value_grouped_left_outer

go run toolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=file_order_item_date_inner
go run toolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=file_order_item_date_left_outer
go run toolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=file_order_date_value_grouped_inner
go run toolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=file_order_date_value_grouped_left_outer
set +x
popd

duration=$SECONDS
echo "$(($duration / 60))m $(($duration % 60))s elapsed."
