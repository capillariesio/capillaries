#!/bin/bash

source ../../common/util.sh

keyspace="global_affairs_quicktest" # to run big: global_affairs_bigtest
cfgDir=/tmp/capi_cfg/global_affairs_quicktest
outDir=/tmp/capi_out/global_affairs_quicktest
scriptFile=$cfgDir/script.json
paramsFile=$cfgDir/script_params_quicktest.json # to run big: script_params_bigtest.json

one_daemon_run $keyspace $scriptFile $paramsFile $outDir '1_read_project_financials'
