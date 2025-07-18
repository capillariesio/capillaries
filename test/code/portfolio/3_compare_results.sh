
#!/bin/bash

source ../common/util.sh

quick_or_big=$1
fs_or_s3=$2

if [[ "$quick_or_big" != "quick" && "$quick_or_big" != "big" || \
  "$fs_or_s3" != "fs" && "$fs_or_s3" != "s3" ]]; then
  echo $(basename "$0") requires 2 parameters:  'quick|big' 'fs|s3'
  exit 1
fi

dataDirName=portfolio_${quick_or_big}test
outDir=/tmp/capi_out/$dataDirName

if [[ "$fs_or_s3" = "s3" ]]; then
  check_s3
  outS3=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_out/$dataDirName
  echo "Downloading files from "$outS3
  if [[ "$quick_or_big" = "big" ]]; then
    aws s3 cp $outS3/account_year_perf.parquet $outDir/
    aws s3 cp $outS3/account_period_sector_perf.parquet $outDir/
  else 
    aws s3 cp $outS3/account_year_perf.csv $outDir
    aws s3 cp $outS3/account_period_sector_perf.csv $outDir/
  fi
fi

if [[ "$quick_or_big" = "big" ]]; then
  cmdDiff="go run ../parquet/capiparquet.go"
  if ! $cmdDiff diff $outDir/account_year_perf.parquet $outDir/account_year_perf_baseline.parquet ||
    ! $cmdDiff diff $outDir/account_period_sector_perf.parquet $outDir/account_period_sector_perf_baseline.parquet; then
    echo -e "\033[0;31m portfolio_bigtest $fs_or_s3 diff FAILED\e[0m"
    exit 1
  else
    echo -e "\033[0;32m portfolio_bigtest $fs_or_s3 diff OK\e[0m"
  fi
else
  if ! diff -b $outDir/account_year_perf.csv $outDir/account_year_perf_baseline.csv ||
    ! diff -b $outDir/account_period_sector_perf.csv $outDir/account_period_sector_perf_baseline.csv; then
    echo -e "\033[0;31m portfolio_quicktest $fs_or_s3 FAILED\e[0m"
    exit 1
  else
    echo -e "\033[0;32m portfolio_quicktest $fs_or_s3 diff OK\e[0m"
  fi
fi