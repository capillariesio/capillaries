#!/bin/bash

source ./util.sh
keyspace="test_tag_and_denormalize"
scriptDir="../../../test/tag_and_denormalize"
outDir="../../../test/tag_and_denormalize/data/out"
scriptFile=$scriptDir/script.json
paramsFile=$scriptDir/script_params_one_run.json

# HTTP(S) script URIs are supported. They slow down things a lot.
#scriptFile=https://github.com/capillariesio/capillaries/blob/main/test/tag_and_denormalize/script.json?raw=1
#paramsFile=https://github.com/capillariesio/capillaries/blob/main/test/tag_and_denormalize/script_params_one_run.json?raw=1

SECONDS=0
[ ! -d "./data/out" ] && mkdir ./data/out
pushd ../../pkg/exe/toolbelt
go run toolbelt.go drop_keyspace -keyspace=$keyspace
go run toolbelt.go start_run -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -start_nodes=read_tags,read_products
echo "Waiting for run to start..."
wait $keyspace 1 1 $outDir
echo "Waiting for run to finish, make sure pkg/exe/daemon is running..."
wait $keyspace 1 2 $outDir
go run toolbelt.go get_node_history -keyspace=$keyspace -run_ids=1
popd
duration=$SECONDS
echo "$(($duration / 60))m $(($duration % 60))s elapsed."