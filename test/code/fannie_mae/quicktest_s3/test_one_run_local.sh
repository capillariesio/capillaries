#!/bin/bash

echo "Make sure that pkg/exe/toolbelt has access to Cassandra and RabbitMQ"

./4_clean_local.sh
./1_copy_data.sh
./2_one_run_local.sh
if ! ./3_compare_results.sh; then
  echo "NOT CLEANED"
  exit 1
else
  ./4_clean_local.sh
fi
