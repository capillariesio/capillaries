#!/bin/bash

source ../../common/util.sh
check_s3

keyspace="portfolio_bigtest_cloud"
scriptFile=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_cfg/portfolio_bigtest/script.json
paramsFile=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_cfg/portfolio_bigtest/script_params_aws.json # Use script_params_cassandra.json if running against Cassandra
startNodes=1_read_accounts,1_read_txns,1_read_period_holdings

# Do not run locally (takes forever)
# Run in the cloud:
# 527s on 4 x c7gd.4xlarge Cassandra nodes, 16 cores each
# 179s on 4 x c7gd.16xlarge Cassandra nodes, 64 cores each
check_cloud_deployment
one_daemon_run_webapi 'http://'$BASTION_IP':'$EXTERNAL_WEBAPI_PORT $keyspace $scriptFile $paramsFile $startNodes
