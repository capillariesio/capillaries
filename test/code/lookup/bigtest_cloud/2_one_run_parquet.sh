#!/bin/bash

source ../../common/util.sh
check_s3

keyspace="lookup_bigtest"
outDir=/tmp/capi_out/lookup_bigtest
scriptFile=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_cfg/lookup_bigtest/script_parquet.json
paramsFile=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_cfg/lookup_bigtest/script_params_one_run.json
startNodes="read_orders,read_order_items"

# Run locally (2000s on my laptop)
#one_daemon_run $keyspace $scriptFile $paramsFile $outDir 'read_orders,read_order_items'

# Run in the cloud (23s on 4 x c7gd.4xlarge Cassandra nodes, 16 cores each)
check_cloud_deployment
one_daemon_run_webapi 'http://'$BASTION_IP':'$EXTERNAL_WEBAPI_PORT $keyspace $scriptFile $paramsFile $startNodes
