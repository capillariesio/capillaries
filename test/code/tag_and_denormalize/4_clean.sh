#!/bin/bash

outDir=/tmp/capi_out/tag_and_denormalize_quicktest

rm -f $outDir/tag_totals.tsv $outDir/tagged_products_for_operator_review.csv $outDir/runs.csv
pushd ../../../pkg/exe/toolbelt
  go run capitoolbelt.go drop_keyspace -keyspace=tag_and_denormalize_quicktest
popd