#!/bin/bash

source ../common/util.sh

keyspace="test_tag_and_denormalize"
# rootUrl=https://github.com/capillariesio/capillaries/blob/main/test/
# scriptFile=$rootUrl/cfg/tag_and_denormalize/script.json?raw=1
scriptFile=$dataDir/cfg/tag_and_denormalize/script.json
# paramsFile=$rootUrl/cfg/tag_and_denormalize/script_params_one_run.json?raw=1
paramsFile=$dataDir/cfg/tag_and_denormalize/script_params_one_run.json
outDir=$dataDir/out/tag_and_denormalize

one_daemon_run  $keyspace $scriptFile $paramsFile $outDir 'read_tags,read_products'