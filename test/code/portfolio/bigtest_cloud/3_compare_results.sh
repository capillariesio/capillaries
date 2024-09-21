#!/bin/bash

outS3=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_out/portfolio_bigtest
outDir=/tmp/capi_out/portfolio_bigtest

echo "Downloading files from "$outS3
aws s3 cp $outS3/account_year_perf.parquet $outDir/
aws s3 cp $outS3/account_period_sector_perf.parquet $outDir/

cmdDiff="go run ../../parquet/capiparquet.go"
if ! $cmdDiff diff $outDir/account_year_perf.parquet $outDir/account_year_perf_baseline.parquet ||
  ! $cmdDiff diff $outDir/account_period_sector_perf.parquet $outDir/account_period_sector_perf_baseline.parquet; then
  echo -e "\033[0;31m portfolio_bigtest diff FAILED\e[0m"
  exit 1
else
  echo -e "\033[0;32m portfolio_bigtest diff OK\e[0m"
fi
