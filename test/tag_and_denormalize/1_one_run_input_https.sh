#!/bin/bash

source ./util.sh

keyspace="test_tag_and_denormalize"

#cfgDir="../../../test/tag_and_denormalize"
cfgDir=https://github.com/capillariesio/capillaries/blob/main/test/tag_and_denormalize
#scriptFile=$cfgDir/script.json
scriptFile=$cfgDir/script.json?raw=1
#paramsFile=$cfgDir/script_params_one_run_input_https.json
paramsFile=$cfgSir/script_params_one_run_input_https.json?raw=1

outDir="../../../test/tag_and_denormalize/data/out"

one_run $keyspace $scriptFile $paramsFile $outDir
