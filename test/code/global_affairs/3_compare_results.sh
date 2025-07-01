#!/bin/bash

source ../common/util.sh

quick_or_big=$1
fs_or_s3=$2

if [[ "$quick_or_big" != "quick" && "$quick_or_big" != "big" || \
  "$fs_or_s3" != "fs" && "$fs_or_s3" != "s3" ]]; then
  echo $(basename "$0") requires 2 parameters:  'quick|big' 'fs|s3'
  exit 1
fi

dataDirName=global_affairs_${quick_or_big}test
outDir=/tmp/capi_out/$dataDirName

if [[ "$fs_or_s3" = "s3" ]]; then
  check_s3
  outS3=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_out/$dataDirName
  echo "Downloading files from "$outS3
  aws s3 cp $outS3/project_country_quarter_amt.csv $outDir/
  aws s3 cp $outS3/project_partner_quarter_amt.csv $outDir/
  aws s3 cp $outS3/project_sector_quarter_amt.csv $outDir/
fi

if [[ "$quick_or_big" = "quick" ]]; then
  if ! diff -b $outDir/project_country_quarter_amt_baseline.csv $outDir/project_country_quarter_amt.csv ||
    ! diff -b $outDir/project_partner_quarter_amt_baseline.csv $outDir/project_partner_quarter_amt.csv ||
    ! diff -b $outDir/project_sector_quarter_amt_baseline.csv $outDir/project_sector_quarter_amt.csv; then
    echo -e "\033[0;31m global_affairs_quicktest $fs_or_s3 diff FAILED\e[0m"
    exit 1
  else
    echo -e "\033[0;32m global_affairs_quicktest $fs_or_s3 diff OK\e[0m"
  fi
else
  if [[ -e $outDir/project_country_quarter_amt.csv && -e $outDir/project_partner_quarter_amt.csv && -e $outDir/project_sector_quarter_amt.csv ]]; then
    echo -e "\033[0;32m global_affairs_quicktest $fs_or_s3 diff OK (result present, not compared)\e[0m"
  else
    echo -e "\033[0;31m global_affairs_bigtest $fs_or_s3 diff FAILED (result missing)\e[0m"
    exit 1
  fi
fi