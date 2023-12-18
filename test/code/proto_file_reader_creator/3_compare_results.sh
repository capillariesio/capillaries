#!/bin/bash

cfgDir=/tmp/capi_cfg/proto_file_reader_creator_quicktest
inDir=/tmp/capi_in/proto_file_reader_creator_quicktest
outDir=/tmp/capi_out/proto_file_reader_creator_quicktest

# Compare CSV
if ! diff -b $inDir/products.csv $outDir/products.csv; then
  echo -e "CSV comparison \033[0;31mFAILED\e[0m"
  exit 1
else
  echo -e "CSV comparison \033[0;32mOK\e[0m"
fi

# Compare parquet
cmdDiff="go run ../parquet/capiparquet.go"
if ! $cmdDiff diff $inDir/products.parquet $outDir/products.parquet; then
  echo -e "Parquet comparison \033[0;31mFAILED\e[0m"
  exit 1
else
  echo -e "Parquet comparison \033[0;32mOK\e[0m"
fi

