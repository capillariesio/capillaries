#!/bin/bash

source ../../common/util.sh
check_s3

keyspace="lookup_quicktest_s3"
cfgS3=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_cfg/lookup_quicktest
outS3=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_out/lookup_quicktest
scriptFile=$cfgS3/script.json
paramsFile=$cfgS3/script_params_one_run_s3.json
outDir=/tmp/capi_out/lookup_quicktest

one_daemon_run $keyspace $scriptFile $paramsFile $outDir 'read_orders,read_order_items'
