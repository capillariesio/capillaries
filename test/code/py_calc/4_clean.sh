#!/bin/bash

inDir=/tmp/capi_in/py_calc_quicktest
outDir=/tmp/capi_out/py_calc_quicktest

rm -f $inDir/raw $inDir/header $inDir/data*
rm -f $outDir/raw $outDir/header $outDir/data* $outDir/*0.csv $outDir/*1.csv $outDir/*py.csv $outDir/*go.csv
pushd ../../../pkg/exe/toolbelt
  go run capitoolbelt.go drop_keyspace -keyspace=py_calc_quicktest_json
  go run capitoolbelt.go drop_keyspace -keyspace=py_calc_quicktest_yaml
popd