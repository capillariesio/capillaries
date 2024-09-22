#!/bin/bash

# Assumptions:
# - this script is run from test/code/py_calc
# - python interpreter is available by name 'python' (see environment config files capidaemon.json and capitoolbelt.json)

source ../common/util.sh

keyspace="py_calc_quicktest_yaml"
cfgDir=/tmp/capi_cfg/py_calc_quicktest
outDir=/tmp/capi_out/py_calc_quicktest
scriptFile=$cfgDir/script.yaml
paramsFile=$cfgDir/script_params.yaml

one_daemon_run  $keyspace $scriptFile $paramsFile $outDir 'read_order_items'
