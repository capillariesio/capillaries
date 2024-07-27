#!/bin/bash

source ../../common/util.sh
check_s3

keyspace="lookup_quicktest_s3"
scriptFile=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_cfg/lookup_quicktest/script.json
paramsFile=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_cfg/lookup_quicktest/script_params_one_run_s3.json
startNodes="read_orders,read_order_items"

# Run in the cloud (527s on 4 x c7gd.4xlarge Cassandra nodes, 16 cores each)
check_cloud_deployment
one_daemon_run_webapi 'http://'$BASTION_IP':'$EXTERNAL_WEBAPI_PORT $keyspace $scriptFile $paramsFile $startNodes
