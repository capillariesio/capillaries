#!/bin/bash

fs_or_https=$1
one_or_multi=$2

if [[ "$fs_or_https" != "fs" && "$fs_or_https" != "https" || \
  "$one_or_multi" != "one" && "$one_or_multi" != "multi" ]]; then
  echo $(basename "$0") requires 2 parameters: 'fs|https' 'one|multi'
  exit 1
fi

dataDirName=tag_and_denormalize_quicktest
keyspace=${dataDirName}_${fs_or_https}_${one_or_multi}

outDir=/tmp/capi_out/$dataDirName

rm -f $outDir/tag_totals.tsv $outDir/tagged_products_for_operator_review.csv $outDir/runs.csv
pushd ../../../pkg/exe/toolbelt
  go run capitoolbelt.go drop_keyspace -keyspace=$keyspace
popd