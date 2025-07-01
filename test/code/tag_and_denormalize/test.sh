#!/bin/bash

fs_or_https=$1
one_or_multi=$2

if [[ "$fs_or_https" != "fs" && "$fs_or_https" != "https" || \
  "$one_or_multi" != "one" && "$one_or_multi" != "multi" ]]; then
  echo $(basename "$0") requires 2 parameters: 'fs|https' 'one|multi'
  exit 1
fi

./4_clean.sh $fs_or_https $one_or_multi
./1_copy_data.sh
./2_run.sh $fs_or_https $one_or_multi
if ! ./3_compare_results.sh; then
  echo "NOT CLEANED"
  exit 1
else
  ./4_clean.sh $fs_or_https $one_or_multi
fi
