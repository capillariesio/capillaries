#!/bin/bash

source ../../common/util.sh

keyspace="fannie_mae_quicktest"
scriptFile=/tmp/capi_cfg/fannie_mae_quicktest/script.json
paramsFile=/tmp/capi_cfg/fannie_mae_quicktest/script_params.json
outDir=/tmp/capi_out/fannie_mae_quicktest

one_daemon_run  $keyspace $scriptFile $paramsFile $outDir '01_read_payments'
