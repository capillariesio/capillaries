#!/bin/bash

pushd ./test/code/lookup
# 13 s
./test.sh quick local fs one
# 10 s
./test.sh quick local fs multi
# 201 s
./test.sh big local fs one
# 183 s
./test.sh big local fs multi
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

pushd ./test/code/portfolio
# 33 s
./test.sh quick local fs multi
popd

pushd ./test/code/proto_file_reader_creator
# 7 s
./test.sh
popd

pushd ./test/code/fannie_mae
# one 218 s
./test.sh quick local fs multi
popd

pushd ./test/code/global_affairs
# one 55 s
./test.sh quick local fs multi
popd

# https, not ready yet

# pushd ./test/code/tag_and_denormalize
# ./test.sh https one
# ./test.sh https multi
# popd
