#!/bin/bash

echo "Make sure that pkg/exe/toolbelt has access to Cassandra and RabbitMQ"

./4_clean.sh quick local fs one
./1_copy_data.sh quick fs
./2_run.sh quick local fs one
if ! ./3_compare_results.sh quick fs; then
  echo "NOT CLEANED"
  exit 1
else
  ./4_clean.sh quick local fs one
fi
