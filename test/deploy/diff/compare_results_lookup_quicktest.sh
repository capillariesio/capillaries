#!/bin/bash

outDir=/mnt/capi_out/lookup_quicktest
if ! diff -b $outDir/order_item_date_inner_baseline.csv $outDir/order_item_date_inner.csv ||
  ! diff -b $outDir/order_item_date_left_outer_baseline.csv $outDir/order_item_date_left_outer.csv ||
  ! diff -b $outDir/order_date_value_grouped_inner_baseline.csv $outDir/order_date_value_grouped_inner.csv ||
  ! diff -b $outDir/order_date_value_grouped_left_outer_baseline.csv $outDir/order_date_value_grouped_left_outer.csv; then
  echo -e "lookup_quicktest diff \033[0;31mFAILED\e[0m"
  exit 1
else
  echo -e "lookup_quicktest \033[0;32mOK\e[0m"
fi