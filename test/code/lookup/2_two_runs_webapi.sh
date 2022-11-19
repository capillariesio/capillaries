#!/bin/bash

source ../common/util.sh

keyspace="test_lookup"
scriptFile=/tmp/capitest_cfg/lookup/script.json
paramsFile=/tmp/capitest_cfg/lookup/script_params_two_runs.json
outDir=/tmp/capitest_out/lookup

two_daemon_runs_webapi $keyspace $scriptFile $paramsFile $outDir 'read_orders,read_order_items' 'order_item_date_inner,order_item_date_left_outer,order_date_value_grouped_inner,order_date_value_grouped_left_outer'