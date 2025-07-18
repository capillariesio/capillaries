#!/bin/bash

keyspace="lookup_quicktest_local_fs_execnodes"
scriptFile=/tmp/capi_cfg/lookup_quicktest/script_quick.yaml
paramsFile=/tmp/capi_cfg/lookup_quicktest/script_params_quick_fs_one.yaml

SECONDS=0

pushd ../../../pkg/exe/toolbelt
set -x

go run capitoolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=read_orders
go run capitoolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=read_order_items

go run capitoolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=order_item_date_inner
go run capitoolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=order_item_date_left_outer
go run capitoolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=order_date_value_grouped_inner
go run capitoolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=order_date_value_grouped_left_outer

go run capitoolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=file_order_item_date_inner
go run capitoolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=file_order_item_date_left_outer
go run capitoolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=file_order_date_value_grouped_inner
go run capitoolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=file_order_date_value_grouped_left_outer

set +x
popd

duration=$SECONDS
echo "$duration s elapsed"
