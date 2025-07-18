#!/bin/bash

keyspace="portfolio_quicktest_local_fs_execnodes"
scriptFile=/tmp/capi_cfg/portfolio_quicktest/script_quick.json
paramsFile=/tmp/capi_cfg/portfolio_quicktest/script_params_quick_fs_one.json

SECONDS=0

pushd ../../../pkg/exe/toolbelt
set -x

go run capitoolbelt.go drop_keyspace -keyspace=$keyspace
go run capitoolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=1_read_accounts
go run capitoolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=1_read_txns
go run capitoolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=2_account_txns_outer
go run capitoolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=1_read_period_holdings
go run capitoolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=2_account_period_holdings_outer
go run capitoolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=3_build_account_period_activity
CAPI_PYCALC_INTERPRETER_PATH=python go run capitoolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=4_calc_account_period_perf
go run capitoolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=5_tag_by_period
go run capitoolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=5_tag_by_sector
CAPI_PYCALC_INTERPRETER_PATH=python go run capitoolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=6_perf_json_to_columns
go run capitoolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=7_file_account_period_sector_perf
go run capitoolbelt.go exec_node -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -node_id=7_file_account_year_perf

set +x

popd

duration=$SECONDS
echo "$duration s elapsed"
