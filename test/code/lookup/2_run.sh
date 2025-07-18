#!/bin/bash

source ../common/util.sh

quick_or_big=$1
local_or_cloud=$2
fs_or_s3=$3
one_or_multi=$4

if [[ "$quick_or_big" != "quick" && "$quick_or_big" != "big" || \
  "$local_or_cloud" != "local" && "$local_or_cloud" != "cloud" || \
  "$fs_or_s3" != "fs" && "$fs_or_s3" != "s3" || \
  "$one_or_multi" != "one" && "$one_or_multi" != "multi" ]]; then
  echo $(basename "$0") requires 4 parameters:  'quick|big' 'local|cloud' 'fs|s3' 'one|multi'
  exit 1
fi

if [[ "$local_or_cloud" = "cloud" ]]; then
  check_cloud_deployment
fi

dataDirName=lookup_${quick_or_big}test

keyspace=${dataDirName}_${local_or_cloud}_${fs_or_s3}_${one_or_multi}

if [[ "$fs_or_s3" = "s3" ]]; then
  check_s3
  if [[ "$quick_or_big" = "big" ]]; then
    scriptFile=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_cfg/$dataDirName/script_big.json
    paramsFile=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_cfg/$dataDirName/script_params_big_s3_${one_or_multi}.json
  else
    scriptFile=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_cfg/$dataDirName/script_quick.yaml
    paramsFile=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_cfg/$dataDirName/script_params_quick_s3_${one_or_multi}.yaml
  fi
else
  if [[ "$quick_or_big" = "big" ]]; then
    scriptFile=/tmp/capi_cfg/$dataDirName/script_big.json
    paramsFile=/tmp/capi_cfg/$dataDirName/script_params_big_fs_${one_or_multi}.json
  else
    scriptFile=/tmp/capi_cfg/$dataDirName/script_quick.yaml
    paramsFile=/tmp/capi_cfg/$dataDirName/script_params_quick_fs_${one_or_multi}.yaml
  fi
fi

startNodes1="read_orders,read_order_items"
if [[ "$one_or_multi" = "multi" ]]; then
  startNodes2='order_item_date_inner,order_item_date_left_outer,order_date_value_grouped_inner,order_date_value_grouped_left_outer'
fi

if [[ "$local_or_cloud" = "cloud" ]]; then
  webapi_multi_run 'http://'$BASTION_IP':'$EXTERNAL_WEBAPI_PORT $keyspace $scriptFile $paramsFile $startNodes1 $startNodes2
else
  webapi_multi_run 'http://localhost:6543' $keyspace $scriptFile $paramsFile $startNodes1 $startNodes2
fi