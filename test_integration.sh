#!/bin/bash

pushd ./test/code/lookup/quicktest
./test_one_run.sh
popd

pushd ./test/code/lookup/quicktest
./test_two_runs.sh
popd

pushd ./test/code/lookup/quicktest
./test_two_runs_webapi.sh
popd

pushd ./test/code/py_calc
./test_one_run.sh
popd

pushd ./test/code/tag_and_denormalize
./test_one_run.sh
popd

pushd ./test/code/tag_and_denormalize
./test_two_runs.sh
popd

pushd ./test/code/tag_and_denormalize
./test_one_run_input_https.sh
popd

pushd ./test/code/portfolio/quicktest
./test_one_run.sh
popd

pushd ./test/code/proto_file_reader_creator
./test_one_run.sh
popd

pushd ./test/code/fannie_mae/quicktest
./test_one_run.sh
popd

