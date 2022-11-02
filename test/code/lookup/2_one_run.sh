#!/bin/bash

source ../common/util.sh

keyspace="test_lookup"
dataDir="../../../test/data"
outDir=$dataDir/out/lookup
scriptFile=$dataDir/cfg/lookup/script.json
paramsFile=$dataDir/cfg/lookup/script_params_one_run.json

one_daemon_run  $keyspace $scriptFile $paramsFile $outDir 'read_orders,read_order_items'
