#!/bin/bash

source ../../common/util.sh

keyspace="fannie_mae_bigtest"
scriptFile=/tmp/capi_cfg/fannie_mae_bigtest/script.json
paramsFile=/tmp/capi_cfg/fannie_mae_bigtest/script_params.json
outDir=/tmp/capi_out/fannie_mae_bigtest

one_daemon_run  $keyspace $scriptFile $paramsFile $outDir '01_read_payments'
