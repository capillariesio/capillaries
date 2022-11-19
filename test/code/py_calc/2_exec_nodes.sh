#!/bin/bash

# Assumptions:
# - this script is run from test/code/py_calc
# - python interpreter is available by name 'python' (see env_config.json)

keyspace="test_py_calc"
outDir=/tmp/capitest_out/py_calc
scriptFile=/tmp/capitest_cfg/py_calc/script.json
paramsFile=/tmp/capitest_cfg/py_calc/script_params.json

SECONDS=0

pushd ../../../pkg/exe/toolbelt
set -x

go run toolbelt.go drop_keyspace -keyspace=$keyspace
go run toolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=read_order_items
go run toolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=taxed_order_items_py
go run toolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=file_taxed_order_items_py
go run toolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=taxed_order_items_go
go run toolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=file_taxed_order_items_go

set +x
popd

duration=$SECONDS
echo "$(($duration / 60))m $(($duration % 60))s elapsed."
