#!/bin/bash

echo "Make sure that pkg/exe/toolbelt has access to Cassandra and RabbitMQ"

./4_clean_s3.sh
./1_copy_data_s3.sh
./2_one_run_s3.sh
if ! ./3_compare_results_s3.sh; then
  echo "NOT CLEANED"
  exit 1
else
  ./4_clean_s3.sh
fi
