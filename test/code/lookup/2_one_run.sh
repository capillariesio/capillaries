#!/bin/bash

source ../common/util.sh

keyspace="lookup_quicktest"
cfgDir=/tmp/capi_cfg/lookup_quicktest
outDir=/tmp/capi_out/lookup_quicktest
scriptFile=$cfgDir/script.json
paramsFile=$cfgDir/script_params_one_run.json

one_daemon_run $keyspace $scriptFile $paramsFile $outDir 'read_orders,read_order_items'
