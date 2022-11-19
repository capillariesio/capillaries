#!/bin/bash

inDir=/tmp/capitest_in/py_calc
outDir=/tmp/capitest_out/py_calc

rm -f $inDir/raw $inDir/header $inDir/data*
rm -f $outDir/raw $outDir/header $outDir/data* $outDir/*0.csv $outDir/*1.csv $outDir/*py.csv $outDir/*go.csv
pushd ../../../pkg/exe/toolbelt
  go run toolbelt.go drop_keyspace -keyspace=test_py_calc
popd