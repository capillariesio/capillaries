#!/bin/bash

source ../../common/util.sh

keyspace="lookup_quicktest_one_run_webapi"
cfgDir=/tmp/capi_cfg/lookup_quicktest
outDir=/tmp/capi_out/lookup_quicktest
scriptFile=$cfgDir/script.json
paramsFile=$cfgDir/script_params_one_run.json
startNodes=read_orders,read_order_items

one_daemon_run_webapi 'http://localhost:6543' $keyspace $scriptFile $paramsFile $startNodes 