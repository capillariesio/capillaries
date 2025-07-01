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
  echo $(basename "$0") requires 4 parameters:  'quick|big' 'local|cloud' 'fs|s3' 'one|multi|execnodes'
  exit 1
fi

if [[ "$local_or_cloud" = "cloud" ]]; then
  check_cloud_deployment
fi

dataDirName=fannie_mae_${quick_or_big}test
outDir=/tmp/capi_out/$dataDirName

if [[ "$quick_or_big" = "big" ]]; then
  rm -f $outDir/deal_seller_summaries.parquet $outDir/deal_summaries.parquet
else
  rm -f $outDir/deal_seller_summaries.parquet $outDir/deal_summaries.parquet $outDir/loan_smrs_clcltd.parquet
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
  # Wait with deleting, it costs too much to upload those?
  # aws s3 rm $inS3/  --recursive
  aws s3 rm $outS3/  --recursive
fi


