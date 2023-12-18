#!/bin/bash

source ../common/util.sh

keyspace="proto_file_reader_creator_parquet_quicktest"
cfgDir=/tmp/capi_cfg/proto_file_reader_creator_quicktest
outDir=/tmp/capi_out/proto_file_reader_creator_quicktest
scriptFile=$cfgDir/script_parquet.json

one_daemon_run_no_params $keyspace $scriptFile $outDir 'read_file'
