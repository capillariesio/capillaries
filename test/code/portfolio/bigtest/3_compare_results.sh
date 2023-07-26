#!/bin/bash

outDir=/tmp/capi_out/portfolio_bigtest
cmdDiff="go run ../../parquet/capiparquet.go"
if ! $cmdDiff diff $outDir/account_year_perf.parquet $outDir/account_year_perf_baseline.parquet ||
  ! $cmdDiff diff $outDir/account_period_sector_perf.parquet $outDir/account_period_sector_perf_baseline.parquet; then
  echo -e "\033[0;31mdiff FAILED\e[0m"
  exit 1
else
  echo -e "\033[0;32mdiff OK\e[0m"
fi
