#!/bin/bash

cfgDir=/tmp/capi_cfg/proto_file_reader_creator_quicktest
inDir=/tmp/capi_in/proto_file_reader_creator_quicktest
outDir=/tmp/capi_out/proto_file_reader_creator_quicktest

rm -f $inDir/* $cfgDir/* $outDir/*
pushd ../../../pkg/exe/toolbelt
  echo "Dropping keyspaces..."
  go run capitoolbelt.go drop_keyspace -keyspace=proto_file_reader_creator_csv_quicktest
  go run capitoolbelt.go drop_keyspace -keyspace=proto_file_reader_creator_parquet_quicktest
popd