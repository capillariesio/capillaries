#!/bin/bash

outDir=/tmp/capi_out/lookup_quicktest

if ! diff -b $outDir/order_item_date_inner_baseline.csv $outDir/order_item_date_inner.csv ||
  ! diff -b $outDir/order_item_date_left_outer_baseline.csv $outDir/order_item_date_left_outer.csv ||
  ! diff -b $outDir/order_date_value_grouped_inner_baseline.csv $outDir/order_date_value_grouped_inner.csv ||
  ! diff -b $outDir/order_date_value_grouped_left_outer_baseline.csv $outDir/order_date_value_grouped_left_outer.csv; then
  echo -e "\033[0;31m lookup_quicktest local fs diff FAILED\e[0m"
  exit 1
else
  echo -e "\033[0;32m lookup_quicktest local fs diff OK\e[0m"
fi