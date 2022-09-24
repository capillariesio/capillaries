#!/bin/bash

# Assumptions:
# - this script is run from test/py_calc
# - python interpreter is available by name 'python'

keyspace="test_py_calc"
scriptDir="../../../test/py_calc"
outDir="../../../test/py_calc/data/out"
scriptFile=$scriptDir/script.json
paramsFile=$scriptDir/script_params.json

# HTTP(S) script URIs are supported. They slow down things a lot.
#scriptFile=https://github.com/kleineshertz/capillaries/blob/main/test/py_calc/script.json?raw=1
#paramsFile=https://github.com/kleineshertz/capillaries/blob/main/test/py_calc/script_params.json?raw=1

SECONDS=0
[ ! -d "./data/out" ] && mkdir ./data/out
pushd ../../pkg/exe/toolbelt
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
