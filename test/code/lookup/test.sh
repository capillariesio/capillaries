#!/bin/bash

quick_or_big=$1
local_or_cloud=$2
fs_or_s3=$3
one_or_multi=$4

if [[ "$quick_or_big" != "quick" && "$quick_or_big" != "big" || \
  "$local_or_cloud" != "local" && "$local_or_cloud" != "cloud" || \
  "$fs_or_s3" != "fs" && "$fs_or_s3" != "s3" || \
  "$one_or_multi" != "one" && "$one_or_multi" != "multi" ]]; then
  echo $(basename "$0") requires 4 parameters:  'quick|big' 'local|cloud' 'fs|s3' 'one|multi'
  exit 1
fi

./4_clean.sh $quick_or_big $local_or_cloud $fs_or_s3 $one_or_multi
./1_create_data.sh $quick_or_big $fs_or_s3
./2_run.sh $quick_or_big $local_or_cloud $fs_or_s3 $one_or_multi
sleep 5 # Parquet files may take a while
if ! ./3_compare_results.sh $quick_or_big $fs_or_s3; then
  echo "NOT CLEANED"
  exit 1
else
  ./4_clean.sh $quick_or_big $local_or_cloud $fs_or_s3 $one_or_multi
fi
