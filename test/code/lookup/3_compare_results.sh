
#!/bin/bash

source ../common/util.sh

quick_or_big=$1
fs_or_s3=$2
csv_or_parquet=$3

if [[ "$quick_or_big" != "quick" && "$quick_or_big" != "big" || \
  "$fs_or_s3" != "fs" && "$fs_or_s3" != "s3" || \
  "$csv_or_parquet" != "csv" && "$csv_or_parquet" != "parquet" ]]; then
  echo $(basename "$0") requires 3 parameters:  'quick|big' 'fs|s3' 'csv|parquet'
  exit 1
fi

dataDirName=lookup_${quick_or_big}test
outDir=/tmp/capi_out/$dataDirName

if [[ "$fs_or_s3" = "s3" ]]; then
  check_s3
  outS3=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_out/$dataDirName
  echo "Downloading files from "$outS3
  if [[ "$quick_or_big" = "big" ]]; then
    aws s3 cp $outS3/order_item_date_inner.parquet $outDir/
    aws s3 cp $outS3/order_item_date_left_outer.parquet $outDir/
    aws s3 cp $outS3/order_date_value_grouped_inner.parquet $outDir/
    aws s3 cp $outS3/order_date_value_grouped_left_outer.parquet $outDir/
  else 
    if [[ "$csv_or_parquet" = "csv" ]]; then
      aws s3 cp $outS3/order_item_date_inner.csv $outDir/
      aws s3 cp $outS3/order_item_date_left_outer.csv $outDir/
      aws s3 cp $outS3/order_date_value_grouped_inner.csv $outDir/
      aws s3 cp $outS3/order_date_value_grouped_left_outer.csv $outDir/
    else 
      aws s3 cp $outS3/order_item_date_inner.parquet $outDir/
      aws s3 cp $outS3/order_item_date_left_outer.parquet $outDir/
      aws s3 cp $outS3/order_date_value_grouped_inner.parquet $outDir/
      aws s3 cp $outS3/order_date_value_grouped_left_outer.parquet $outDir/
    fi
  fi
fi

if [[ "$quick_or_big" = "big" ]]; then
  cmdDiff="go run ../parquet/capiparquet.go"
  if ! $cmdDiff diff $outDir/order_item_date_inner_baseline.parquet $outDir/order_item_date_inner.parquet ||
    ! $cmdDiff diff $outDir/order_item_date_left_outer_baseline.parquet $outDir/order_item_date_left_outer.parquet ||
    ! $cmdDiff diff $outDir/order_date_value_grouped_inner_baseline.parquet $outDir/order_date_value_grouped_inner.parquet ||
    ! $cmdDiff diff $outDir/order_date_value_grouped_left_outer_baseline.parquet $outDir/order_date_value_grouped_left_outer.parquet; then
    echo -e "\033[0;31m lookup_bigtest parquet $fs_or_s3 diff FAILED\e[0m"
    exit 1
  else
    echo -e "\033[0;32m lookup_bigtest parquet $fs_or_s3 diff OK\e[0m"
  fi
else
  if [[ "$csv_or_parquet" = "csv" ]]; then
    if ! diff -b $outDir/order_item_date_inner_baseline.csv $outDir/order_item_date_inner.csv ||
      ! diff -b $outDir/order_item_date_left_outer_baseline.csv $outDir/order_item_date_left_outer.csv ||
      ! diff -b $outDir/order_date_value_grouped_inner_baseline.csv $outDir/order_date_value_grouped_inner.csv ||
      ! diff -b $outDir/order_date_value_grouped_left_outer_baseline.csv $outDir/order_date_value_grouped_left_outer.csv; then
      echo -e "\033[0;31m lookup_quicktest csv $fs_or_s3 diff FAILED\e[0m"
      exit 1
    else
      echo -e "\033[0;32m lookup_quicktest csv $fs_or_s3 diff OK\e[0m"
    fi
  else
    cmdDiff="go run ../parquet/capiparquet.go"
    if ! $cmdDiff diff $outDir/order_item_date_inner_baseline.parquet $outDir/order_item_date_inner.parquet ||
      ! $cmdDiff diff $outDir/order_item_date_left_outer_baseline.parquet $outDir/order_item_date_left_outer.parquet ||
      ! $cmdDiff diff $outDir/order_date_value_grouped_inner_baseline.parquet $outDir/order_date_value_grouped_inner.parquet ||
      ! $cmdDiff diff $outDir/order_date_value_grouped_left_outer_baseline.parquet $outDir/order_date_value_grouped_left_outer.parquet; then
      echo -e "\033[0;31m lookup_bigtest parquet $fs_or_s3 diff FAILED\e[0m"
      exit 1
    else
      echo -e "\033[0;32m lookup_quicktest parquet $fs_or_s3 diff OK\e[0m"
    fi
  fi
fi