#!/bin/bash

# Assumptions:
# - this script is run from test/code/py_calc
# - python interpreter is available by name 'python' (see environment config files capidaemon.json and capitoolbelt.json)

source ../common/util.sh

json_or_yaml=$1

if [[ "$json_or_yaml" != "json" && "$json_or_yaml" != "yaml" ]]; then
  echo $(basename "$0") requires 1 parameter: 'json|yaml'
  exit 1
fi

dataDirName=py_calc_quicktest
keyspace=${dataDirName}_${json_or_yaml}

cfgDir=/tmp/capi_cfg/${dataDirName}
outDir=/tmp/capi_out/${dataDirName}
scriptFile=$cfgDir/script.json
paramsFile=$cfgDir/script_params.json

webapi_multi_run 'http://localhost:6543' $keyspace $scriptFile $paramsFile read_order_items
