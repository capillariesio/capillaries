#!/bin/bash

source ../common/util.sh

fs_or_https=$1
one_or_multi=$2

if [[ "$fs_or_https" != "fs" && "$fs_or_https" != "https" || \
  "$one_or_multi" != "one" && "$one_or_multi" != "multi" ]]; then
  echo $(basename "$0") requires 2 parameters: 'fs|https' 'one|multi'
  exit 1
fi

dataDirName=tag_and_denormalize_quicktest
keyspace=${dataDirName}_${fs_or_https}_${one_or_multi}

if [[ "$fs_or_https" = "https" ]]; then
  rootUrl=https://raw.githubusercontent.com/capillariesio/capillaries/main/test/data
  scriptFile=$rootUrl/cfg/tag_and_denormalize/script.json
  paramsFile=$rootUrl/cfg/tag_and_denormalize/script_params_https_${one_or_multi}.json
else
  scriptFile=/tmp/capi_cfg/$dataDirName/script.json
  paramsFile=/tmp/capi_cfg/$dataDirName/script_params_fs_${one_or_multi}.json
fi

startNodes1='read_tags,read_products'
if [[ "$one_or_multi" = "multi" ]]; then
  startNodes2='tag_totals'
fi

if [[ "$local_or_cloud" = "cloud" ]]; then
  webapi_multi_run 'http://'$BASTION_IP':'$EXTERNAL_WEBAPI_PORT $keyspace $scriptFile $paramsFile $startNodes1 $startNodes2
else
  webapi_multi_run 'http://localhost:6543' $keyspace $scriptFile $paramsFile $startNodes1 $startNodes2
fi
