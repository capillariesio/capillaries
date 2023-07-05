#!/bin/bash

outDir=/mnt/capi_out/portfolio_quicktest

if ! diff -b $outDir/account_year_perf.csv $outDir/account_year_perf_baseline.csv ||
  ! diff -b $outDir/account_period_sector_perf.csv $outDir/account_period_sector_perf_baseline.csv; then
  echo -e "portfolio_quicktest diff \033[0;31mFAILED\e[0m"
  exit 1
else
  echo -e "portfolio_quicktest diff \033[0;32mOK\e[0m"
fi