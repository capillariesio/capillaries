#!/bin/bash

source ../common/util.sh

keyspace="test_lookup"
outDir=/tmp/capitest_out/lookup
scriptFile=/tmp/capitest_cfg/lookup/script.json
paramsFile=/tmp/capitest_cfg/lookup/script_params_one_run.json

one_daemon_run $keyspace $scriptFile $paramsFile $outDir 'read_orders,read_order_items'
