#!/bin/bash

export outDir=/tmp/capi_out/fannie_mae_quicktest
export cmdDiff="go run ../../parquet/capiparquet.go"

if ! $cmdDiff diff $outDir/deal_seller_summaries_baseline.parquet $outDir/deal_seller_summaries.parquet ||
  ! $cmdDiff diff $outDir/deal_servicer_summaries_baseline.parquet $outDir/deal_servicer_summaries.parquet ||
  ! $cmdDiff diff $outDir/deal_summaries_baseline.parquet $outDir/deal_summaries.parquet ||
  ! $cmdDiff diff $outDir/loan_summaries_calculated_baseline.parquet $outDir/loan_summaries_calculated.parquet; then
  echo -e "\033[0;31mdiff FAILED\e[0m"
  exit 1
else
  echo -e "\033[0;32mdiff OK\e[0m"
fi
