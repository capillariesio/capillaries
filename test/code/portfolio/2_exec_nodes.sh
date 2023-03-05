#!/bin/bash

keyspace="portfolio_quicktest"
scriptFile=/tmp/capi_cfg/portfolio_quicktest/script.json
paramsFile=/tmp/capi_cfg/portfolio_quicktest/script_params.json

SECONDS=0

pushd ../../../pkg/exe/toolbelt
set -x

go run capitoolbelt.go drop_keyspace -keyspace=$keyspace
go run capitoolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=read_accounts
go run capitoolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=read_txns
go run capitoolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=account_txns_outer
go run capitoolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=read_period_holdings
go run capitoolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=account_period_holdings_outer
go run capitoolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=build_account_period_activity
go run capitoolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=calc_account_period_perf

set +x

popd

duration=$SECONDS
echo "$(($duration / 60))m $(($duration % 60))s elapsed."
