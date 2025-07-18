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

dataDirName=fannie_mae_${quick_or_big}test

keyspace=${dataDirName}_${local_or_cloud}_${fs_or_s3}_${one_or_multi}

if [[ "$fs_or_s3" = "s3" ]]; then
  check_s3
  scriptFile=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_cfg/$dataDirName/script.json
  paramsFile=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_cfg/$dataDirName/script_params_${quick_or_big}_s3_${one_or_multi}.json
else
  scriptFile=/tmp/capi_cfg/$dataDirName/script.json
  paramsFile=/tmp/capi_cfg/$dataDirName/script_params_${quick_or_big}_fs_${one_or_multi}.json
fi

startNodes1="01_read_payments"
if [[ "$one_or_multi" = "multi" ]]; then
  startNodes2='02_loan_ids'
  startNodes3='02_deal_names,02_deal_sellers'
  startNodes4='03_deal_total_upbs,04_loan_payment_summaries,'
  startNodes5='05_deal_summaries,04_loan_smrs_clcltd,05_deal_seller_summaries'
fi

if [[ "$local_or_cloud" = "cloud" ]]; then
  webapi_multi_run 'http://'$BASTION_IP':'$EXTERNAL_WEBAPI_PORT $keyspace $scriptFile $paramsFile $startNodes1 $startNodes2 $startNodes3 $startNodes4 $startNodes5
else
  webapi_multi_run 'http://localhost:6543' $keyspace $scriptFile $paramsFile $startNodes1 $startNodes2 $startNodes3 $startNodes4 $startNodes5
fi