#!/bin/bash

keyspace="tag_and_denormalize_quicktest_execnodes"
scriptFile=/tmp/capi_cfg/tag_and_denormalize_quicktest/script.json
paramsFile=/tmp/capi_cfg/tag_and_denormalize_quicktest/script_params_fs_one.json
outDir=/tmp/capi_out/tag_and_denormalize_quicktest

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
echo "$duration s elapsed"
