#!/bin/bash

inDir=/tmp/capi_in/lookup_bigtest
outDir=/tmp/capi_out/lookup_bigtest

rm -f $outDir/*_inner.csv $outDir/*_outer.csv $outDir/*_inner.parquet $outDir/*_outer.parquet $outDir/runs.csv
pushd ../../../../pkg/exe/toolbelt
  go run capitoolbelt.go drop_keyspace -keyspace=lookup_bigtest
popd