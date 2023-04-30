#!/bin/bash

source ../common/util.sh

keyspace="tag_and_denormalize_quicktest"
scriptFile=/tmp/capi_cfg/tag_and_denormalize_quicktest/script.json
paramsFile=/tmp/capi_cfg/tag_and_denormalize_quicktest/script_params_two_runs.json
outDir=/tmp/capi_out/tag_and_denormalize_quicktest

two_daemon_runs  $keyspace $scriptFile $paramsFile $outDir 'read_tags,read_products' 'tag_totals'
