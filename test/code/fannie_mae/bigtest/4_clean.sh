#!/bin/bash

source ../../common/util.sh
check_s3

cfgS3=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_cfg/fannie_mae_bigtest
inS3=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_in/fannie_mae_bigtest
outS3=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_out/fannie_mae_bigtest

outDir=/tmp/capi_out/fannie_mae_bigtest

rm -f $outDir/*

pushd ../../../../pkg/exe/toolbelt
  go run capitoolbelt.go drop_keyspace -keyspace=fannie_mae_bigtest
popd

aws s3 rm $cfgS3/ --recursive
aws s3 rm $inS3/  --recursive
aws s3 rm $outS3/  --recursive