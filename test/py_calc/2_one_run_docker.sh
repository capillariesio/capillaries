#!/bin/bash

# Assumptions:
# - this script is run from test/py_calc
# - python interpreter is available by name 'python'

source ./util.sh

keyspace="test_py_calc"
scriptDir="/capillaries_src_root/test/py_calc"
outDir="../../../test/py_calc/data/out"
scriptFile=$scriptDir/script.json
paramsFile=$scriptDir/script_params_docker.json

SECONDS=0
[ ! -d "./data/out" ] && mkdir ./data/out
pushd ../../pkg/exe/toolbelt
go run toolbelt.go drop_keyspace -keyspace=$keyspace
go run toolbelt.go start_run -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -start_nodes=read_order_items
echo "Waiting for run to start..."
wait $keyspace 1 1 $outDir
echo "Waiting for run to finish, make sure pkg/exe/daemon container is running..."
wait $keyspace 1 2 $outDir
go run toolbelt.go get_node_history -keyspace=$keyspace -run_ids=1
popd
duration=$SECONDS
echo "$(($duration / 60))m $(($duration % 60))s elapsed."
