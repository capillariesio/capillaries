#!/bin/bash

source ../common/util.sh

quick_or_big=$1
fs_or_s3=$2

if [[ "$quick_or_big" != "quick" && "$quick_or_big" != "big" || \
  "$fs_or_s3" != "fs" && "$fs_or_s3" != "s3" ]]; then
  echo $(basename "$0") requires 2 parameters:  'quick|big' 'fs|s3'
  exit 1
fi

dataDirName=fannie_mae_${quick_or_big}test
outDir=/tmp/capi_out/$dataDirName

if [[ "$fs_or_s3" = "s3" ]]; then
  check_s3
  outS3=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_out/$dataDirName
  echo "Downloading files from "$outS3
  if [[ "$quick_or_big" = "big" ]]; then
    aws s3 cp $outS3/deal_seller_summaries.parquet $outDir/
    aws s3 cp $outS3/deal_summaries.parquet $outDir/
  else 
    aws s3 cp $outS3/deal_seller_summaries.parquet $outDir/
    aws s3 cp $outS3/deal_summaries.parquet $outDir/
    aws s3 cp $outS3/loan_smrs_clcltd.parquet $outDir/   
  fi
fi

if [[ "$quick_or_big" = "big" ]]; then
  cmdDiff="go run ../parquet/capiparquet.go"
  if ! $cmdDiff diff $outDir/deal_seller_summaries_baseline.parquet $outDir/deal_seller_summaries.parquet ||
    ! $cmdDiff diff $outDir/deal_summaries_baseline.parquet $outDir/deal_summaries.parquet; then
    echo -e "\033[0;31m fannie_mae_bigtest $fs_or_s3 diff FAILED\e[0m"
    exit 1
  else
    echo -e "\033[0;32m fannie_mae_bigtest $fs_or_s3 diff OK\e[0m"
  fi
else
  if ! $cmdDiff diff $outDir/deal_seller_summaries_baseline.parquet $outDir/deal_seller_summaries.parquet ||
    ! $cmdDiff diff $outDir/loan_smrs_clcltd.parquet $outDir/loan_smrs_clcltd.parquet ||
    ! $cmdDiff diff $outDir/deal_summaries_baseline.parquet $outDir/deal_summaries.parquet; then
    echo -e "\033[0;31m fannie_mae_quicktest $fs_or_s3 FAILED\e[0m"
    exit 1
  else
    echo -e "\033[0;32m fannie_mae_quicktest $fs_or_s3 diff OK\e[0m"
  fi
fi