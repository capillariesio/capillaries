#!/bin/bash

outDir=/tmp/capi_out/portfolio_quicktest

if ! diff -b $outDir/account_year_perf.csv $outDir/account_year_perf_baseline.csv ||
  ! diff -b $outDir/account_period_sector_perf.csv $outDir/account_period_sector_perf_baseline.csv; then
  echo "FAILED"
  exit 1
else
  echo "SUCCESS"
fi