#!/bin/bash

source ../../common/util.sh

keyspace="lookup_bigtest"
cfgDir=/tmp/capi_cfg/lookup_bigtest
outDir=/tmp/capi_out/lookup_bigtest
scriptFile=$cfgDir/script_parquet.json
paramsFile=$cfgDir/script_params_one_run.json

one_daemon_run $keyspace $scriptFile $paramsFile $outDir 'read_orders,read_order_items'
