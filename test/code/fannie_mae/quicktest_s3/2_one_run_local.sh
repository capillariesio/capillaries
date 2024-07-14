#!/bin/bash

source ../../common/util.sh
check_s3

keyspace="fannie_mae_quicktest_s3"
scriptFile=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_cfg/fannie_mae_quicktest/script.json
paramsFile=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_cfg/fannie_mae_quicktest/script_params_s3.json
outDir=/tmp/capi_out/fannie_mae_quicktest

one_daemon_run  $keyspace $scriptFile $paramsFile $outDir '01_read_payments'
