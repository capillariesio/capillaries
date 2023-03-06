#!/bin/bash

source ../common/util.sh

keyspace="portfolio_quicktest"
scriptFile=/tmp/capi_cfg/portfolio_quicktest/script.json
paramsFile=/tmp/capi_cfg/portfolio_quicktest/script_params.json
outDir=/tmp/capi_out/portfolio_quicktest

one_daemon_run  $keyspace $scriptFile $paramsFile $outDir 'read_accounts,read_txns,read_period_holdings'
