#!/bin/bash

source ../../common/util.sh
check_s3

outS3=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_out/lookup_bigtest
outDir=/tmp/capi_out/lookup_bigtest

echo "Downloading files from "$outS3
aws s3 cp $outS3/order_item_date_inner.parquet $outDir/
aws s3 cp $outS3/order_item_date_left_outer.parquet $outDir/
aws s3 cp $outS3/order_date_value_grouped_inner.parquet $outDir/
aws s3 cp $outS3/order_date_value_grouped_left_outer.parquet $outDir/

cmdDiff="go run ../../parquet/capiparquet.go"
if ! $cmdDiff diff $outDir/order_item_date_inner_baseline.parquet $outDir/order_item_date_inner.parquet ||
  ! $cmdDiff diff $outDir/order_item_date_left_outer_baseline.parquet $outDir/order_item_date_left_outer.parquet ||
  ! $cmdDiff diff $outDir/order_date_value_grouped_inner_baseline.parquet $outDir/order_date_value_grouped_inner.parquet ||
  ! $cmdDiff diff $outDir/order_date_value_grouped_left_outer_baseline.parquet $outDir/order_date_value_grouped_left_outer.parquet; then
  echo -e "\033[0;31m lookup_bigtest parquet diff FAILED\e[0m"
  exit 1
else
  echo -e "\033[0;32m lookup_bigtest parquet diff OK\e[0m"
fi