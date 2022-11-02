#!/bin/bash

dataDir=../../data
inDir=$dataDir/in/lookup
outDir=$dataDir/out/lookup

rm -f $inDir/raw $inDir/header $inDir/data* $inDir/*.csv
rm -f $outDir/raw $outDir/header $outDir/data $outDir/*.csv
pushd ../../../pkg/exe/toolbelt
  go run toolbelt.go drop_keyspace -keyspace=test_lookup
popd