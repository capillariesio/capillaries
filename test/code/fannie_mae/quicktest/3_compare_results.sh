#!/bin/bash

outDir=/tmp/capi_out/fannie_mae_quicktest

if ! diff -b $outDir/distinct_loan_ids.csv $outDir/distinct_loan_ids_baseline.csv; then
  echo -e "\033[0;31mdiff FAILED\e[0m"
  exit 1
else
  echo -e "\033[0;32mdiff OK\e[0m"
fi