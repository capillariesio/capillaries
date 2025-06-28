#!/bin/bash

source ../common/util.sh

quick_or_big=$1
local_or_cloud=$2
fs_or_s3=$3
one_or_multi=$4

if [[ "$quick_or_big" != "quick" && "$quick_or_big" != "big" || "$local_or_cloud" != "local" && "$local_or_cloud" != "cloud" || "$fs_or_s3" != "fs" && "$fs_or_s3" != "s3" || "$one_or_multi" != "one" && "$one_or_multi" != "multi" ]]; then
  echo $(basename "$0") requires 4 parameters:  'quick|big' 'local|cloud' 'fs|s3' 'one|multi'
  exit 1
fi

if [[ "$local_or_cloud" = "cloud" ]]; then
  check_cloud_deployment
fi

if [[ "$quick_or_big" = "big" ]]; then
  dataDirName=portfolio_bigtest
else
  dataDirName=portfolio_quicktest
fi

keyspace=${dataDirName}_${local_or_cloud}_${fs_or_s3}_${one_or_multi}

if [[ "$fs_or_s3" = "s3" ]]; then
  check_s3
  scriptFile=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_cfg/$dataDirName/script_$quick_or_big.json
  paramsFile=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_cfg/$dataDirName/script_params_${quick_or_big}_s3_${one_or_multi}.json
else
  scriptFile=/tmp/capi_cfg/$dataDirName/script_$quick_or_big.json
  paramsFile=/tmp/capi_cfg/$dataDirName/script_params_${quick_or_big}_fs_${one_or_multi}.json
fi

startNodes1='1_read_accounts,1_read_txns,1_read_period_holdings'
if [[ "$one_or_multi" = "multi" ]]; then
  startNodes2='2_account_txns_outer,2_account_period_holdings_outer'
  startNodes3='3_build_account_period_activity'
fi

if [[ "$local_or_cloud" = "cloud" ]]; then
  webapi_multi_run 'http://'$BASTION_IP':'$EXTERNAL_WEBAPI_PORT $keyspace $scriptFile $paramsFile $startNodes1 $startNodes2 $startNodes3
else
  webapi_multi_run 'http://localhost:6543' $keyspace $scriptFile $paramsFile $startNodes1 $startNodes2 $startNodes3
fi