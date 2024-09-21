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
./test_one_run.sh
popd

pushd ./test/code/proto_file_reader_creator
./test_one_run.sh
popd

pushd ./test/code/fannie_mae/quicktest_local_fs
# This will take a few min
./test_one_run.sh
popd

# https

pushd ./test/code/tag_and_denormalize
./test_one_run_input_https.sh
popd

# s3

# These will require something like
# export CAPILLARIES_AWS_TESTBUCKET=capillaries-testbucket
# export AWS_ACCESS_KEY_ID=...
# export AWS_SECRET_ACCESS_KEY=...
# export AWS_DEFAULT_REGION=us-east-1
# for webapi and daemon. Make sure you set those either:
# - before running docker containers with webapi and daemon
# - before executing webapi/daemon from the cmd line or from the debugger 

pushd ./test/code/lookup/quicktest_s3
./test_one_run_local.sh
popd

pushd ./test/code/fannie_mae/quicktest_s3
# This will take a few min
./test_one_run_local.sh
popd

