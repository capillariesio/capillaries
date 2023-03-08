#!/bin/bash

source ../common/util.sh

keyspace="tag_and_denormalize_quicktest"

rootUrl=https://github.com/capillariesio/capillaries/blob/main/test/data
scriptFile=$rootUrl/cfg/tag_and_denormalize_quicktest/script.json?raw=1
paramsFile=$rootUrl/cfg/tag_and_denormalize_quicktest/script_params_one_run_input_https.json?raw=1

outDir=/tmp/capi_out/tag_and_denormalize_quicktest

one_daemon_run  $keyspace $scriptFile $paramsFile $outDir 'read_tags,read_products'