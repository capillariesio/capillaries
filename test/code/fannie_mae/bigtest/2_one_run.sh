#!/bin/bash

source ../../common/util.sh

keyspace="fannie_mae_bigtest"
scriptFile=/tmp/capi_cfg/fannie_mae_bigtest/script.json
outDir=/tmp/capi_out/fannie_mae_bigtest

one_daemon_run_no_params  $keyspace $scriptFile $outDir 'read_file'
