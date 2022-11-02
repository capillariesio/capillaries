#!/bin/bash

echo "Make sure that pkg/exe/toolbelt has access to Cassandra"

./3_clean.sh
./1_exec_nodes.sh
if ! ./2_compare_results.sh; then
  echo "NOT CLEANED"
  exit 1
else
  ./3_clean.sh
fi
