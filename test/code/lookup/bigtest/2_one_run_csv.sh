#!/bin/bash

source ../../common/util.sh
check_s3

keyspace="lookup_bigtest"
outDir=/tmp/capi_out/lookup_bigtest
scriptFile=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_cfg/lookup_bigtest/script_csv.json
paramsFile=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_cfg/lookup_bigtest/script_params_one_run.json

# Run locally (2000s on my laptop)
# one_daemon_run $keyspace $scriptFile $paramsFile $outDir 'read_orders,read_order_items'

# Run in the cloud (23s on 4 x c7gd.4xlarge Cassandra nodes, 16 cores each)
check_cloud_deployment
ssh -o StrictHostKeyChecking=no -i $CAPIDEPLOY_SSH_PRIVATE_KEY_PATH ubuntu@$BASTION_IP "~/bin/capitoolbelt start_run -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -start_nodes=read_orders,read_order_items"
