#!/bin/bash

outDir=/tmp/capi_out/fannie_mae_quicktest

rm -f $outDir/distinct_loan_ids.csv
pushd ../../../pkg/exe/toolbelt
  go run capitoolbelt.go drop_keyspace -keyspace=fannie_mae_quicktest
popd