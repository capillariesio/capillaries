#!/bin/bash

outDir=/mnt/capi_out/portfolio_bigtest
cmdDiff=~/bin/capiparquet
if ! $cmdDiff diff $outDir/account_year_perf.parquet $outDir/account_year_perf_baseline.parquet ||
  ! $cmdDiff diff $outDir/account_period_sector_perf.parquet $outDir/account_period_sector_perf_baseline.parquet; then
  echo -e "portfolio_bigtest diff \033[0;31mFAILED\e[0m"
  exit 1
else
  echo -e "portfolio_bigtest diff \033[0;32mOK\e[0m"
fi
