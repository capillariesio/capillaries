#!/bin/bash

source ../../common/util.sh
check_s3

echo "Make sure that pkg/exe/toolbelt has access to Cassandra and RabbitMQ"

./4_clean.sh
./1_create_data.sh
./2_one_run_parquet.sh
if ! ./3_compare_results_parquet.sh; then
  echo "NOT CLEANED"
  exit 1
else
  ./4_clean.sh
fi
