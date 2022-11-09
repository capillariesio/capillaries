#!/bin/bash

source ../common/util.sh

keyspace="test_lookup"
dataDir="../../../test/data"
outDir=$dataDir/out/lookup
scriptFile=$dataDir/cfg/lookup/script.json
paramsFile=$dataDir/cfg/lookup/script_params_two_runs.json

# Regarding webapi calls via curl:
# WSL2 users cannot reference the app running on the host by loalhost or 127.0.0.1 because of this WSL2 behavior:
# - https://github.com/microsoft/WSL/issues/5211
# - https://superuser.com/questions/1679757/how-to-access-windows-localhost-from-wsl2
# Either use host IP address or the $(localhost).local trick.
# Also, Make sure Windows Firewall does not block incoming connections for this port.

pushd ../../../pkg/exe/toolbelt
[ ! -d $outDir ] && mkdir $outDir

go run toolbelt.go drop_keyspace -keyspace=$keyspace

# Operator starts run 1

curl -d '{"script_uri":"'"$scriptFile"'", "script_params_uri":"'"$paramsFile"'", "start_nodes":"read_orders,read_order_items"}' -H "Content-Type: application/json" -X POST "http://$(hostname).local:6543/ks/$keyspace/run"

echo "Waiting for run to start..."
wait $keyspace 1 1 $outDir
echo "Waiting for run to finish, make sure pkg/exe/daemon is running..."
wait $keyspace 1 2 $outDir
go run toolbelt.go get_node_history -keyspace=$keyspace -run_ids=1

# Operator approves intermediate results and starts run 2

curl -d '{"script_uri":"'"$scriptFile"'", "script_params_uri":"'"$paramsFile"'", "start_nodes":"order_item_date_inner,order_item_date_left_outer,order_date_value_grouped_inner,order_date_value_grouped_left_outer"}' -H "Content-Type: application/json" -X POST "http://$(hostname).local:6543/ks/$keyspace/run"

echo "Waiting for run to start..."
wait $keyspace 2 1 $outDir
echo "Waiting for run to finish, make sure pkg/exe/daemon is running..."
wait $keyspace 2 2 $outDir
curl "http://$(hostname).local:6543/ks/$keyspace"
popd
