#!/bin/bash

source ../../common/util.sh
check_s3

outS3=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_out/lookup_bigtest
outDir=/tmp/capi_out/lookup_bigtest

echo "Downloading files from "$outS3
aws s3 cp $outS3/order_item_date_inner.csv $outDir/
aws s3 cp $outS3/order_item_date_left_outer.csv $outDir/
aws s3 cp $outS3/order_date_value_grouped_inner.csv $outDir/
aws s3 cp $outS3/order_date_value_grouped_left_outer.csv $outDir/

if ! diff -b $outDir/order_item_date_inner_baseline.csv $outDir/order_item_date_inner.csv ||
  ! diff -b $outDir/order_item_date_left_outer_baseline.csv $outDir/order_item_date_left_outer.csv ||
  ! diff -b $outDir/order_date_value_grouped_inner_baseline.csv $outDir/order_date_value_grouped_inner.csv ||
  ! diff -b $outDir/order_date_value_grouped_left_outer_baseline.csv $outDir/order_date_value_grouped_left_outer.csv; then
  echo -e "\033[0;31m lookup_bigtest csv diff FAILED\e[0m"
  exit 1
else
  echo -e "\033[0;32m lookup_bigtest csv diff OK\e[0m"
fi