#!/bin/bash

keyspace="test_tag_and_denormalize"
scriptFile=/tmp/capitest_cfg/tag_and_denormalize/script.json
paramsFile=/tmp/capitest_cfg/tag_and_denormalize/script_params_one_run.json
outDir=/tmp/capitest_out/tag_and_denormalize

SECONDS=0

pushd ../../../pkg/exe/toolbelt
set -x

go run capitoolbelt.go drop_keyspace -keyspace=$keyspace
go run capitoolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=read_tags
go run capitoolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=read_products
go run capitoolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=tag_products
go run capitoolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=file_tagged_products_for_operator_review
go run capitoolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=tag_totals
go run capitoolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=file_tag_totals

set +x
popd

duration=$SECONDS
echo "$(($duration / 60))m $(($duration % 60))s elapsed."
