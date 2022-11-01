#!/bin/bash

source ./util.sh

keyspace="test_tag_and_denormalize"

cfgDir="../../../test/tag_and_denormalize"
scriptFile=$cfgDir/script.json
paramsFile=$cfgDir/script_params_one_run.json

outDir="../../../test/tag_and_denormalize/data/out"

one_run $keyspace $scriptFile $paramsFile $outDir
