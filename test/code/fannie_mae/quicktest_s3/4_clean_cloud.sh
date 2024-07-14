#!/bin/bash

source ../../common/util.sh
check_s3

keyspace="fannie_mae_quicktest_s3"
cfgS3=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_cfg/fannie_mae_quicktest
inS3=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_in/fannie_mae_quicktest
outS3=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_out/fannie_mae_quicktest

outDir=/tmp/capi_out/fannie_mae_quicktest

rm -f $outDir/*

check_cloud_deployment
drop_keyspace_webapi 'http://'$BASTION_IP':'$CAPIDEPLOY_EXTERNAL_WEBAPI_PORT $keyspace

aws s3 rm $cfgS3/ --recursive
aws s3 rm $inS3/  --recursive
aws s3 rm $outS3/  --recursive