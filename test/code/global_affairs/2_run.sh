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

dataDirName=global_affairs_${quick_or_big}test

keyspace=${dataDirName}_${local_or_cloud}_${fs_or_s3}_${one_or_multi}

if [[ "$fs_or_s3" = "s3" ]]; then
  check_s3
  scriptFile=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_cfg/$dataDirName/script.json
  paramsFile=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_cfg/$dataDirName/script_params_${quick_or_big}_s3_${one_or_multi}.json
else
  scriptFile=/tmp/capi_cfg/$dataDirName/script.json
  paramsFile=/tmp/capi_cfg/$dataDirName/script_params_${quick_or_big}_fs_${one_or_multi}.json
fi

startNodes1="1_read_project_financials"
if [[ "$one_or_multi" = "multi" ]]; then
  startNodes2='2_calc_quarterly_budget'
  startNodes3='3_tag_countries,3_tag_sectors'
  startNodes4='4_tag_countries_quarter,4_tag_sectors_quarter,4_tag_partners_quarter'
  startNodes5='5_project_country_quarter_amt,5_project_sector_quarter_amt,5_project_partner_quarter_amt'
fi

if [[ "$local_or_cloud" = "cloud" ]]; then
  webapi_multi_run 'http://'$BASTION_IP':'$EXTERNAL_WEBAPI_PORT $keyspace $scriptFile $paramsFile $startNodes1 $startNodes2 $startNodes3 $startNodes4 $startNodes5
else
  webapi_multi_run 'http://localhost:6543' $keyspace $scriptFile $paramsFile $startNodes1 $startNodes2 $startNodes3 $startNodes4 $startNodes5
fi