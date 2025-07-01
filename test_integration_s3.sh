#!/bin/bash

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

pushd ./test/code/lookup
# 10 s
./test.sh quick local s3 multi
popd

pushd ./test/code/fannie_mae
# 98 s
./test.sh quick local s3 multi
popd
