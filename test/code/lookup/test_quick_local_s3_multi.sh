#!/bin/bash

echo "Make sure that pkg/exe/toolbelt has access to Cassandra and RabbitMQ"

./4_clean.sh quick local s3 multi
./1_create_data.sh quick s3
./2_run.sh quick local s3 multi
if ! ./3_compare_results.sh quick s3; then
  echo "NOT CLEANED"
  exit 1
else
  ./4_clean.sh quick local s3 multi
fi
