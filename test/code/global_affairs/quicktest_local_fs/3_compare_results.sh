#!/bin/bash

export outDir=/tmp/capi_out/global_affairs_quicktest

if ! diff -b $outDir/project_country_quarter_amt_baseline.csv $outDir/project_country_quarter_amt.csv ||
  ! diff -b $outDir/project_partner_quarter_amt_baseline.csv $outDir/project_partner_quarter_amt.csv ||
  ! diff -b $outDir/project_sector_quarter_amt_baseline.csv $outDir/project_sector_quarter_amt.csv; then
  echo -e "\033[0;31m global_affairs_quicktest diff FAILED\e[0m"
  exit 1
else
  echo -e "\033[0;32m global_affairs_quicktest diff OK\e[0m"
fi
