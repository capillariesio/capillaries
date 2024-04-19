#!/bin/bash

pushd ./test/code/lookup/quicktest_s3
./test_one_run_s3.sh
popd

pushd ./test/code/fannie_mae/quicktest_s3
./test_one_run_s3.sh
popd

