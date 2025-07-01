#!/bin/bash

json_or_yaml=$1

if [[ "$json_or_yaml" != "json" && "$json_or_yaml" != "yaml" ]]; then
  echo $(basename "$0") requires 1 parameter: 'json|yaml'
  exit 1
fi

dataDirName=py_calc_quicktest
keyspace=${dataDirName}_${json_or_yaml}

inDir=/tmp/capi_in/$dataDirName
outDir=/tmp/capi_out/$dataDirName

rm -f $inDir/raw $inDir/header $inDir/data*
rm -f $outDir/raw $outDir/header $outDir/data* $outDir/*0.csv $outDir/*1.csv $outDir/*py.csv $outDir/*go.csv
pushd ../../../pkg/exe/toolbelt
  go run capitoolbelt.go drop_keyspace -keyspace=$keyspace
  go run capitoolbelt.go drop_keyspace -keyspace=$keyspace
popd