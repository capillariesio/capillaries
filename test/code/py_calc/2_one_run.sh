#!/bin/bash

# Assumptions:
# - this script is run from test/code/py_calc
# - python interpreter is available by name 'python' (see env_config.json)

source ../common/util.sh

keyspace="test_py_calc"
dataDir="../../../test/data"
outDir=$dataDir/out/py_calc
scriptFile=$dataDir/cfg/py_calc/script.json
paramsFile=$dataDir/cfg/py_calc/script_params.json

one_daemon_run  $keyspace $scriptFile $paramsFile $outDir 'read_order_items'
