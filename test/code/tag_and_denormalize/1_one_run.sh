#!/bin/bash

source ../common/util.sh

keyspace="test_tag_and_denormalize"
scriptFile=/tmp/capitest_cfg/tag_and_denormalize/script.json
paramsFile=/tmp/capitest_cfg/tag_and_denormalize/script_params_one_run.json
outDir=/tmp/capitest_out/tag_and_denormalize

one_daemon_run  $keyspace $scriptFile $paramsFile $outDir 'read_tags,read_products'
