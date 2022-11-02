#!/bin/bash

source ../common/util.sh

keyspace="test_lookup"
dataDir="../../../test/data"
outDir=$dataDir/out/lookup
scriptFile=$dataDir/cfg/lookup/script.json
paramsFile=$dataDir/cfg/lookup/script_params_two_runs.json

two_daemon_runs  $keyspace $scriptFile $paramsFile $outDir 'read_orders,read_order_items' 'order_item_date_inner,order_item_date_left_outer,order_date_value_grouped_inner,order_date_value_grouped_left_outer'


# source ./util.sh
# keyspace="test_lookup"
# scriptDir="../../../test/lookup"
# outDir="../../../test/lookup/data/out"
# scriptFile=$scriptDir/script.json
# paramsFile=$scriptDir/script_params_two_runs.json

# SECONDS=0
# [ ! -d "../../data/out/lookup" ] && mkdir ../../data/out/lookup
# pushd ../../pkg/exe/toolbelt
# go run toolbelt.go drop_keyspace -keyspace=$keyspace

# # Operator starts run 1

# go run toolbelt.go start_run -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -start_nodes=read_orders,read_order_items
# echo "Waiting for run to start..."
# wait $keyspace 1 1 $outDir
# echo "Waiting for run to finish, make sure pkg/exe/daemon is running..."
# wait $keyspace 1 2 $outDir
# go run toolbelt.go get_node_history -keyspace=$keyspace -run_ids=1

# # Operator approves intermediate results and starts run 2

# go run toolbelt.go start_run -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -start_nodes=order_item_date_inner,order_item_date_left_outer,order_date_value_grouped_inner,order_date_value_grouped_left_outer
# echo "Waiting for run to start..."
# wait $keyspace 2 1 $outDir
# echo "Waiting for run to finish, make sure pkg/exe/daemon is running..."
# wait $keyspace 2 2 $outDir
# go run toolbelt.go get_node_history -keyspace=$keyspace -run_ids=1,2
# popd
# duration=$SECONDS
# echo "$(($duration / 60))m $(($duration % 60))s elapsed."
