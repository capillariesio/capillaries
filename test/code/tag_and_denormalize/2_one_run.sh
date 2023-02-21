#!/bin/bash

source ../common/util.sh

keyspace="tag_and_denormalize_quicktest"
scriptFile=/tmp/capi_cfg/tag_and_denormalize_quicktest/script.json
paramsFile=/tmp/capi_cfg/tag_and_denormalize_quicktest/script_params_one_run.json
outDir=/tmp/capi_out/tag_and_denormalize_quicktest

one_daemon_run  $keyspace $scriptFile $paramsFile $outDir 'read_tags,read_products'
