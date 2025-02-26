#!/bin/bash

outDir=/tmp/capi_out/global_affairs_quicktest

pushd ../../../../pkg/exe/toolbelt
  go run capitoolbelt.go drop_keyspace -keyspace=global_affairs_quicktest
popd
rm -f $outDir/*_amt.csv $outDir/runs.csv
