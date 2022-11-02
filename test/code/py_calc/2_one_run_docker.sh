#!/bin/bash

# Assumptions:
# - this script is run from test/code/py_calc
# - python interpreter is available by name 'python' (see env_config.json)

source ../common/util.sh

keyspace="test_py_calc"
scriptFile=/capillaries_docker_test_cfg/py_calc/script.json
paramsFile=/capillaries_docker_test_cfg/py_calc/script_params_docker.json
outDir=/capillaries_docker_test_out/py_calc

one_daemon_run  $keyspace $scriptFile $paramsFile $outDir 'read_order_items'
