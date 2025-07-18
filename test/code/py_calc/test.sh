#!/bin/bash

json_or_yaml=$1

if [[ "$json_or_yaml" != "json" && "$json_or_yaml" != "yaml" ]]; then
  echo $(basename "$0") requires 1 parameter: 'json|yaml'
  exit 1
fi

./4_clean.sh $json_or_yaml
./1_copy_data.sh
./2_run.sh $json_or_yaml
if ! ./3_compare_results.sh; then
  echo "NOT CLEANED"
  exit 1
else
  ./4_clean.sh $json_or_yaml
fi
