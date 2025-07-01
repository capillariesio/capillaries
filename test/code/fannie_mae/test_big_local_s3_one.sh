#!/bin/bash

echo "Make sure that pkg/exe/toolbelt has access to Cassandra and RabbitMQ"

./4_clean.sh big local s3 one
./1_create_data.sh big s3
./2_run.sh big local s3 one
if ! ./3_compare_results.sh big s3; then
  echo "NOT CLEANED"
  exit 1
else
  ./4_clean.sh big local s3 one
fi
