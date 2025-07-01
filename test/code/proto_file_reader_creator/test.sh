#!/bin/bash

./4_clean.sh
./1_generate_scripts.sh
./2_one_run_csv.sh
./2_one_run_parquet.sh
if ! ./3_compare_results.sh; then
  echo "NOT CLEANED"
  exit 1
else
  ./4_clean.sh
fi
