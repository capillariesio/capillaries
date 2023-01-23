#!/bin/bash

echo "Make sure that pkg/exe/toolbelt has access to Cassandra and RabbitMQ"

./4_clean.sh
./1_create_quicktest_data.sh
./2_two_runs.sh
if ! ./3_compare_results.sh; then
  echo "NOT CLEANED"
  exit 1
else
  ./4_clean.sh
fi
