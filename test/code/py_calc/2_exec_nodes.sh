#!/bin/bash

# Assumptions:
# - this script is run from test/code/py_calc
# - python interpreter is available by name 'python' (see environment config files capidaemon.json and capitoolbelt.json)

keyspace="py_calc_quicktest"
cfgDir=/tmp/capi_cfg/py_calc_quicktest
outDir=/tmp/capi_out/py_calc_quicktest
scriptFile=$cfgDir/script.json
paramsFile=$cfgDir/script_params.json

SECONDS=0

pushd ../../../pkg/exe/toolbelt
set -x

go run capitoolbelt.go drop_keyspace -keyspace=$keyspace
go run capitoolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=read_order_items
go run capitoolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=taxed_order_items_py
go run capitoolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=file_taxed_order_items_py
go run capitoolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=taxed_order_items_go
go run capitoolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=file_taxed_order_items_go

set +x
popd

duration=$SECONDS
echo "$(($duration / 60))m $(($duration % 60))s elapsed."
