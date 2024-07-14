#!/bin/bash

source ../../common/util.sh
check_s3

keyspace="portfolio_bigtest"
cfgS3=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_cfg/portfolio_bigtest
inS3=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_in/portfolio_bigtest
outS3=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_out/portfolio_bigtest

outDir=/tmp/capi_out/portfolio_bigtest

rm -f $outDir/account_period_sector_perf.parquet $outDir/account_year_perf.parquet

check_cloud_deployment
drop_keyspace_webapi 'http://'$BASTION_IP':'$CAPIDEPLOY_EXTERNAL_WEBAPI_PORT $keyspace

aws s3 rm $cfgS3/ --recursive
aws s3 rm $inS3/  --recursive
aws s3 rm $outS3/  --recursive