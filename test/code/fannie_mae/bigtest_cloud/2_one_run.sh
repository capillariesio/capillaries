#!/bin/bash

source ../../common/util.sh
check_s3

keyspace="fannie_mae_bigtest"
scriptFile=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_cfg/fannie_mae_bigtest/script.json
paramsFile=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_cfg/fannie_mae_bigtest/script_params.json
startNodes=01_read_payments

# Don't even think about running locally
# Run in the cloud (takes too long on 4 x c7gd.4xlarge Cassandra nodes, 16 cores each - consider a bigger setup)
# 1530s on 4 x c7gd.16xlarge Cassandra nodes, 64 cores each
check_cloud_deployment
one_daemon_run_webapi 'http://'$BASTION_IP':'$EXTERNAL_WEBAPI_PORT $keyspace $scriptFile $paramsFile $startNodes

