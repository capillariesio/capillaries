#!/bin/bash

./4_clean_cloud.sh
./1_copy_data.sh
./2_one_run_cloud.sh
if ! ./3_compare_results.sh; then
  echo "NOT CLEANED"
  exit 1
else
  ./4_clean_cloud.sh
fi
