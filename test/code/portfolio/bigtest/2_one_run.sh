#!/bin/bash

source ../common/util.sh

keyspace="portfolio_bigtest"
scriptFile=/tmp/capi_cfg/portfolio_bigtest/script.json
paramsFile=/tmp/capi_cfg/portfolio_bigtest/script_params.json
outDir=/tmp/capi_out/portfolio_bigtest

one_daemon_run  $keyspace $scriptFile $paramsFile $outDir '1_read_accounts,1_read_txns,1_read_period_holdings'
