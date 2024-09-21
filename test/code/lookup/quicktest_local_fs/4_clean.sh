#!/bin/bash

inDir=/tmp/capi_in/lookup_quicktest
outDir=/tmp/capi_out/lookup_quicktest

rm -f $outDir/*_inner.csv $outDir/*_outer.csv $outDir/*_inner.parquet $outDir/*_outer.parquet $outDir/runs.csv
pushd ../../../../pkg/exe/toolbelt
  go run capitoolbelt.go drop_keyspace -keyspace=lookup_quicktest_exec_nodes
  go run capitoolbelt.go drop_keyspace -keyspace=lookup_quicktest_one_run_webapi
  go run capitoolbelt.go drop_keyspace -keyspace=lookup_quicktest_one_run_json
  go run capitoolbelt.go drop_keyspace -keyspace=lookup_quicktest_one_run_yaml
  go run capitoolbelt.go drop_keyspace -keyspace=lookup_quicktest_two_runs_json
popd