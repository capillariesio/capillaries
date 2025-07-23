#!/bin/bash

short_or_long_or_s3_or_all=$1

if [[ "$short_or_long_or_s3_or_all" != "short" && "$short_or_long_or_s3_or_all" != "long" && "$short_or_long_or_s3_or_all" != "s3" && "$short_or_long_or_s3_or_all" != "all" ]]; then
  echo $(basename "$0") requires 1 parameter: 'short|long|s3|all'
  exit 1
fi

if [[ "$short_or_long_or_s3_or_all" = "short" || "$short_or_long_or_s3_or_all" = "all" ]]; then
	pushd ./test/code/lookup
	# 13 s
	./test.sh quick local fs one
	# 10 s
	./test.sh quick local fs multi
	popd

	pushd ./test/code/py_calc
	# 12 s
	./test.sh json
	# 12 s
	./test.sh yaml
	popd

	pushd ./test/code/tag_and_denormalize
	# 17 s
	./test.sh fs one
	# 7 s
	./test.sh fs multi
	popd

	pushd ./test/code/proto_file_reader_creator
	# 7 s
	./test.sh
	popd

	pushd ./test/code/tag_and_denormalize
	# 22 s
	./test.sh https one
	# 22 s
	./test.sh https multi
	popd
fi

if [[ "$short_or_long_or_s3_or_all" = "long" || "$short_or_long_or_s3_or_all" = "all" ]]; then
	pushd ./test/code/lookup
	# 193 s
	./test.sh big local fs one
	# 203 s
	./test.sh big local fs multi
	popd

	pushd ./test/code/portfolio
	# one 103 s multi 94 s
	./test.sh quick local fs multi
	popd

	pushd ./test/code/fannie_mae
	# one 181 s
	./test.sh quick local fs multi
	popd

	pushd ./test/code/global_affairs
	# one 44 s
	./test.sh quick local fs multi
	popd
fi

if [[ "$short_or_long_or_s3_or_all" = "s3" || "$short_or_long_or_s3_or_all" = "all" ]]; then
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
fi

