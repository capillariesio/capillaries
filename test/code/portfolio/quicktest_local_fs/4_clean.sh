#!/bin/bash

outDir=/tmp/capi_out/portfolio_quicktest

rm -f $outDir/account_period_sector_perf.csv $outDir/account_year_perf.csv
pushd ../../../../pkg/exe/toolbelt
  go run capitoolbelt.go drop_keyspace -keyspace=portfolio_quicktest
popd