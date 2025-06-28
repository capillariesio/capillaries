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

outDir=/tmp/capi_out/$dataDirName

if [[ "$quick_or_big" = "big" ]]; then
  rm -f $outDir/account_period_sector_perf.parquet $outDir/account_year_perf.parquet
else
  rm -f $outDir/account_period_sector_perf.csv $outDir/account_year_perf.csv
fi

keyspace=${dataDirName}_${local_or_cloud}_${fs_or_s3}_${one_or_multi}

if [[ "$local_or_cloud" = "cloud" ]]; then
  drop_keyspace_webapi 'http://'$BASTION_IP':'$EXTERNAL_WEBAPI_PORT $keyspace
else
  drop_keyspace_webapi 'http://localhost:6543' $keyspace
fi

if [[ "$fs_or_s3" = "s3" ]]; then
  check_s3
  cfgS3=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_cfg/$dataDirName
  inS3=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_in/$dataDirName
  outS3=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_out/$dataDirName
  aws s3 rm $cfgS3/ --recursive
  aws s3 rm $inS3/  --recursive
  aws s3 rm $outS3/  --recursive
fi


