#!/bin/bash

source ../../common/util.sh

keyspace="lookup_quicktest"
cfgDir=s3://capillaries-sampledeployment005/capi_cfg/lookup_quicktest
outDir=/tmp/capi_out/lookup_quicktest
scriptFile=$cfgDir/script.json
paramsFile=$cfgDir/script_params_one_run_s3.json

one_daemon_run $keyspace $scriptFile $paramsFile $outDir 'read_orders,read_order_items'
