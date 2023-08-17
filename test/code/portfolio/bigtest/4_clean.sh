#!/bin/bash

outDir=/tmp/capi_out/portfolio_bigtest

rm -f $outDir/account_period_sector_perf.parquet $outDir/account_year_perf.parquet
pushd ../../../../pkg/exe/toolbelt
  go run capitoolbelt.go drop_keyspace -keyspace=portfolio_bigtest
popd