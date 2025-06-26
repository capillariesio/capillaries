#!/bin/bash

pushd ./test/code/lookup/quicktest_local_fs
./test_one_run.sh
./test_one_run_yaml.sh
./test_two_runs.sh
./test_one_run_webapi.sh
popd

pushd ./test/code/py_calc
./test_one_run_json.sh
./test_one_run_yaml.sh
popd

pushd ./test/code/tag_and_denormalize
./test_one_run.sh
./test_two_runs.sh
popd

pushd ./test/code/portfolio/quicktest_local_fs
./test_three_runs.sh
popd

pushd ./test/code/proto_file_reader_creator
./test_one_run.sh
popd

pushd ./test/code/fannie_mae/quicktest_local_fs
# This will take a few min
./test_one_run.sh
popd

pushd ./test/code/global_affairs/quicktest_local_fs
# This will take 1 min
./test_one_run.sh
popd

# https

pushd ./test/code/tag_and_denormalize
./test_one_run_input_https.sh
popd

 exit 0

# s3

# The section below requires 3 things.

# 1. Capillaries binaries (daemon and webapi) need AWS credentials to access S3 artifacts 
# export AWS_ACCESS_KEY_ID=...
# export AWS_SECRET_ACCESS_KEY=...
# export AWS_DEFAULT_REGION=us-east-1
# Make sure you set those either:
# - before running docker containers with webapi and daemon, so docker-compose.yml will pass those variables to the binaries
# - before executing webapi/daemon from the cmd line or from the debugger 

# 2. The commands below need same AWS credentials to access S3 artifacts, so set those variables before running this script.

# 3. The commands below need to know the location of S3 artifacts, so, before running this script, specify something like
# export CAPILLARIES_AWS_TESTBUCKET=capillaries-testbucket

pushd ./test/code/lookup/quicktest_s3
./test_one_run_local.sh
popd

pushd ./test/code/fannie_mae/quicktest_s3
# This will take a few min
./test_one_run_local.sh
popd

