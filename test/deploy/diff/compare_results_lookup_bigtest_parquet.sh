#!/bin/bash

outDir=/mnt/capi_out/lookup_bigtest/
cmdDiff=~/bin/capiparquet
if ! $cmdDiff diff $outDir/order_item_date_inner_baseline.parquet $outDir/order_item_date_inner.parquet ||
  ! $cmdDiff diff $outDir/order_item_date_left_outer_baseline.parquet $outDir/order_item_date_left_outer.parquet ||
  ! $cmdDiff diff $outDir/order_date_value_grouped_inner_baseline.parquet $outDir/order_date_value_grouped_inner.parquet ||
  ! $cmdDiff diff $outDir/order_date_value_grouped_left_outer_baseline.parquet $outDir/order_date_value_grouped_left_outer.parquet; then
  echo -e "\033[0;31mdiff FAILED\e[0m"
  exit 1
else
  echo -e "\033[0;32mdiff OK\e[0m"
fi