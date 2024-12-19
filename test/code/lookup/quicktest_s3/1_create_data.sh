#!/bin/bash

source ../../common/util.sh
check_s3

# Generate in and out-baseline files
../quicktest_local_fs/1_create_data.sh

cfgS3=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_cfg/lookup_quicktest
inS3=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_in/lookup_quicktest
cfgDir=/tmp/capi_cfg/lookup_quicktest
inDir=/tmp/capi_in/lookup_quicktest

echo "Copying config files to "$cfgS3
sed -e 's~$CAPILLARIES_AWS_TESTBUCKET~'$CAPILLARIES_AWS_TESTBUCKET'~g' ../../../data/cfg/lookup_quicktest/script_params_one_run_s3.json > $cfgDir/script_params_one_run_s3.json
aws s3 cp $cfgDir/script.json $cfgS3/
aws s3 cp $cfgDir/script_params_one_run_s3.json $cfgS3/

echo "Copying in files to "$inS3
aws s3 cp $inDir $inS3/ --recursive --include "*.csv"



