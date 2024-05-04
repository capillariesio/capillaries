#!/bin/bash

source ../../common/util.sh
check_s3

keyspace="fannie_mae_bigtest"
outDir=/tmp/capi_out/fannie_mae_quicktest
scriptFile=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_cfg/fannie_mae_bigtest/script.json
paramsFile=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_cfg/fannie_mae_bigtest/script_params.json

# Run locally (don't even think about it)
#one_daemon_run  $keyspace $scriptFile $paramsFile $outDir '01_read_payments'

# Run in the cloud (takes too long on 4 x c7gd.4xlarge Cassandra nodes, 16 cores each - consider a bigger setup)
check_cloud_deployment
ssh -o StrictHostKeyChecking=no -i $CAPIDEPLOY_SSH_PRIVATE_KEY_PATH ubuntu@$BASTION_IP "~/bin/capitoolbelt start_run -script_file=$scriptFile -params_file=$paramsFile -keyspace=$keyspace -start_nodes=01_read_payments"

