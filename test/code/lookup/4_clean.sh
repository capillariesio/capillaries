#!/bin/bash

inDir=/tmp/capitest_in/lookup
outDir=/tmp/capitest_out/lookup

rm -f $inDir/raw $inDir/header $inDir/data*
rm -f $outDir/raw $outDir/header $outDir/data $outDir/*_inner.csv $outDir/*_outer.csv $outDir/runs.csv
pushd ../../../pkg/exe/toolbelt
  go run toolbelt.go drop_keyspace -keyspace=test_lookup
popd