#!/bin/bash

source ../common/util.sh

keyspace="proto_file_reader_creator_csv_quicktest"
cfgDir=/tmp/capi_cfg/proto_file_reader_creator_quicktest
outDir=/tmp/capi_out/proto_file_reader_creator_quicktest
scriptFile=$cfgDir/script_csv.json

one_daemon_run_no_params $keyspace $scriptFile $outDir 'read_file'
