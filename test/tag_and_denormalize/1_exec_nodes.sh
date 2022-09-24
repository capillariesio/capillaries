#!/bin/bash

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
set -x
go run toolbelt.go drop_keyspace -keyspace=$keyspace
go run toolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=read_tags
go run toolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=read_products
go run toolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=tag_products
go run toolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=file_tagged_products
go run toolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=tag_totals
go run toolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=file_tag_totals
set +x
popd
duration=$SECONDS
echo "$(($duration / 60))m $(($duration % 60))s elapsed."
