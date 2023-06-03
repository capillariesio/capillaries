#!/bin/bash

source ../../common/util.sh

keyspace="lookup_quicktest"
cfgDir=/tmp/capi_cfg/lookup_quicktest
outDir=/tmp/capi_out/lookup_quicktest
scriptFile=$cfgDir/script.json
paramsFile=$cfgDir/script_params_two_runs.json

two_daemon_runs_webapi $keyspace $scriptFile $paramsFile $outDir 'read_orders,read_order_items' 'order_item_date_inner,order_item_date_left_outer,order_date_value_grouped_inner,order_date_value_grouped_left_outer'