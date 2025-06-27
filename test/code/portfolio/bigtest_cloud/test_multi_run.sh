#!/bin/bash

source ../../common/util.sh
check_s3

./4_clean.sh
./1_create_data.sh
./2_multi_run.sh
if ! ./3_compare_results.sh; then
  echo "NOT CLEANED"
  exit 1
else
  ./4_clean.sh
fi
