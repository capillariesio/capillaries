#!/bin/bash

outDir=/tmp/capitest_out/tag_and_denormalize

rm -f $outDir/tag_totals.tsv $outDir/tagged_products_for_operator_review.csv $outDir/runs.csv
pushd ../../../pkg/exe/toolbelt
  go run toolbelt.go drop_keyspace -keyspace=test_tag_and_denormalize
popd