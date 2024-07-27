#!/bin/bash

source ../../common/util.sh
check_s3

cfgS3=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_cfg/lookup_quicktest
inS3=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_in/lookup_quicktest
outS3=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_out/lookup_quicktest

inDir=/tmp/capi_in/lookup_quicktest
outDir=/tmp/capi_out/lookup_quicktest

rm -f $outDir/*_inner.csv $outDir/*_outer.csv $outDir/*_inner.parquet $outDir/*_outer.parquet $outDir/runs.csv
pushd ../../../../pkg/exe/toolbelt
  go run capitoolbelt.go drop_keyspace -keyspace=lookup_quicktest_s3
popd

aws s3 rm $cfgS3/ --recursive
aws s3 rm $inS3/  --recursive
aws s3 rm $outS3/  --recursive