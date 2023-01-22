#!/bin/bash

# Assumptions:
# - this script is run from test/code/py_calc
# - python interpreter is available by name 'python' (see environment config files capidaemon.json and capitoolbelt.json)

source ../common/util.sh

keyspace="test_py_calc"
outDir=/tmp/capitest_out/py_calc
scriptFile=/tmp/capitest_cfg/py_calc/script.json
paramsFile=/tmp/capitest_cfg/py_calc/script_params.json

one_daemon_run  $keyspace $scriptFile $paramsFile $outDir 'read_order_items'
