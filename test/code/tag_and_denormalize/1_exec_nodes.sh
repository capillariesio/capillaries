#!/bin/bash

keyspace="test_tag_and_denormalize"

dataDir="../../../test/data"
outDir=$dataDir/out/tag_and_denormalize
scriptFile=$dataDir/cfg/tag_and_denormalize/script.json
paramsFile=$dataDir/cfg/tag_and_denormalize/script_params_one_run.json

SECONDS=0

pushd ../../../pkg/exe/toolbelt
[ ! -d $outDir ] && mkdir $outDir
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
