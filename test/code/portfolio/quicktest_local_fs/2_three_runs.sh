#!/bin/bash

source ../../common/util.sh

keyspace="portfolio_quicktest"
scriptFile=/tmp/capi_cfg/portfolio_quicktest/script.json
paramsFile=/tmp/capi_cfg/portfolio_quicktest/script_params.json
startNodesOne='1_read_accounts,1_read_txns,1_read_period_holdings'
startNodesTwo='2_account_txns_outer,2_account_period_holdings_outer'
startNodesThree='3_build_account_period_activity'
outDir=/tmp/capi_out/portfolio_quicktest

#check_cloud_deployment
#three_daemon_runs_webapi 'http://'$BASTION_IP':'$EXTERNAL_WEBAPI_PORT $keyspace $scriptFile $paramsFile $startNodesOne  $startNodesTwo $startNodesThree
three_daemon_runs_webapi 'http://localhost:6543' $keyspace $scriptFile $paramsFile $startNodesOne  $startNodesTwo $startNodesThree