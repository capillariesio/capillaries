#!/bin/bash

cfgDir=/tmp/capi_cfg/proto_file_reader_creator_quicktest
inDir=/tmp/capi_in/proto_file_reader_creator_quicktest
outDir=/tmp/capi_out/proto_file_reader_creator_quicktest

# Compare CSV
if ! diff -b $inDir/products.csv $outDir/products.csv; then
  echo -e "\033[0;31m proto_file_reader_creator_quicktest CSV diff FAILED\e[0m"
  exit 1
else
  echo -e "\033[0;32m proto_file_reader_creator_quicktest CSV diff OK\e[0m"
fi

# Compare parquet
cmdDiff="go run ../parquet/capiparquet.go"
if ! $cmdDiff diff $inDir/products.parquet $outDir/products.parquet; then
  echo -e "\033[0;31m proto_file_reader_creator_quicktest Parquet diff FAILED\e[0m"
  exit 1
else
  echo -e "\033[0;32m proto_file_reader_creator_quicktest Parquet diff OK\e[0m"
fi

