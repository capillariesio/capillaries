#!/bin/bash

source ./util.sh

keyspace="test_tag_and_denormalize"

cfgDir="../../../test/tag_and_denormalize"
scriptFile=$cfgDir/script.json
paramsFile=$cfgDir/script_params_two_runs.json

outDir="../../../test/tag_and_denormalize/data/out"

two_runs $keyspace $scriptFile $paramsFile $outDir
