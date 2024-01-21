#!/bin/bash

source ../../common/util.sh

keyspace="fannie_mae_quicktest"
scriptFile=/tmp/capi_cfg/fannie_mae_quicktest/script.json
outDir=/tmp/capi_out/fannie_mae_quicktest

one_daemon_run_no_params  $keyspace $scriptFile $outDir 'read_file'
