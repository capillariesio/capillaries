#!/bin/bash

./4_clean.sh quick local fs execnodes
./1_create_data.sh quick fs
./2_exec_nodes.sh
if ! ./3_compare_results.sh quick fs; then
  echo "NOT CLEANED"
  exit 1
else
  ./4_clean.sh quick local fs execnodes
fi
