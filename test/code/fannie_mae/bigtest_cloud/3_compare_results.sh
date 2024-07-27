#!/bin/bash

source ../../common/util.sh
check_s3

outS3=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_out/fannie_mae_bigtest
outDir=/tmp/capi_out/fannie_mae_bigtest
export cmdDiff="go run ../../parquet/capiparquet.go"

aws s3 cp $outS3/deal_seller_summaries.parquet $outDir/
aws s3 cp $outS3/deal_summaries.parquet $outDir/

if ! $cmdDiff diff $outDir/deal_seller_summaries_baseline.parquet $outDir/deal_seller_summaries.parquet ||
  ! $cmdDiff diff $outDir/deal_summaries_baseline.parquet $outDir/deal_summaries.parquet; then
  echo -e "\033[0;31mdiff FAILED\e[0m"
  exit 1
else
  echo -e "\033[0;32mdiff OK\e[0m"
fi