#!/bin/bash

inDir=/tmp/capi_in/lookup_quicktest
outDir=/tmp/capi_out/lookup_quicktest

rm -f $inDir/raw $inDir/header $inDir/data*
rm -f $outDir/raw $outDir/header $outDir/data $outDir/*_inner.csv $outDir/*_outer.csv $outDir/runs.csv
pushd ../../../pkg/exe/toolbelt
  go run capitoolbelt.go drop_keyspace -keyspace=lookup_quicktest
popd