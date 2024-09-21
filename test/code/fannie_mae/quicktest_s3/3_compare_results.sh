#!/bin/bash

source ../../common/util.sh
check_s3

outS3=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_out/fannie_mae_quicktest
outDir=/tmp/capi_out/fannie_mae_quicktest
export cmdDiff="go run ../../parquet/capiparquet.go"

aws s3 cp $outS3/deal_seller_summaries.parquet $outDir/
aws s3 cp $outS3/deal_summaries.parquet $outDir/
aws s3 cp $outS3/loan_summaries_calculated.parquet $outDir/

if ! $cmdDiff diff $outDir/deal_seller_summaries_baseline.parquet $outDir/deal_seller_summaries.parquet ||
  ! $cmdDiff diff $outDir/loan_summaries_calculated_baseline.parquet $outDir/loan_summaries_calculated.parquet ||
  ! $cmdDiff diff $outDir/deal_summaries_baseline.parquet $outDir/deal_summaries.parquet; then
  echo -e "\033[0;31m fannie_mae_quicktest s3 diff FAILED\e[0m"
  exit 1
else
  echo -e "\033[0;32m fannie_mae_quicktest s3 diff OK\e[0m"
fi