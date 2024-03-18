#!/bin/bash

outS3Url=s3://capillaries-sampledeployment005/capi_out/lookup_quicktest
outDir=/tmp/capi_out/lookup_quicktest

aws s3 cp $outS3Url/order_item_date_inner.csv $outDir/
aws s3 cp $outS3Url/order_item_date_left_outer.csv $outDir/
aws s3 cp $outS3Url/order_date_value_grouped_inner.csv $outDir/
aws s3 cp $outS3Url/order_date_value_grouped_left_outer.csv $outDir/

if ! diff -b $outDir/order_item_date_inner_baseline.csv $outDir/order_item_date_inner.csv ||
  ! diff -b $outDir/order_item_date_left_outer_baseline.csv $outDir/order_item_date_left_outer.csv ||
  ! diff -b $outDir/order_date_value_grouped_inner_baseline.csv $outDir/order_date_value_grouped_inner.csv ||
  ! diff -b $outDir/order_date_value_grouped_left_outer_baseline.csv $outDir/order_date_value_grouped_left_outer.csv; then
  echo -e "\033[0;31mdiff FAILED\e[0m"
  exit 1
else
  echo -e "\033[0;32mdiff OK\e[0m"
fi