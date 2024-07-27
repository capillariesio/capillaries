#!/bin/bash

source ../../common/util.sh
check_s3

keyspace="fannie_mae_quicktest_s3"
scriptFile=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_cfg/fannie_mae_quicktest/script.json
paramsFile=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_cfg/fannie_mae_quicktest/script_params_s3.json
startNodes=01_read_payments

# Run in the cloud (46s on 4 x c7gd.4xlarge Cassandra nodes, 16 cores each)
check_cloud_deployment
one_daemon_run_webapi 'http://'$BASTION_IP':'$EXTERNAL_WEBAPI_PORT $keyspace $scriptFile $paramsFile $startNodes

