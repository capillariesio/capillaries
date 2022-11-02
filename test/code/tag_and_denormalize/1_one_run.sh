#!/bin/bash

source ../common/util.sh

keyspace="test_tag_and_denormalize"
dataDir="../../../test/data"
outDir=$dataDir/out/tag_and_denormalize
scriptFile=$dataDir/cfg/tag_and_denormalize/script.json
paramsFile=$dataDir/cfg/tag_and_denormalize/script_params_one_run.json

one_daemon_run  $keyspace $scriptFile $paramsFile $outDir 'read_tags,read_products'
