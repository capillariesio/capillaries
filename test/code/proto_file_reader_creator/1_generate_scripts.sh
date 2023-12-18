#!/bin/bash

cfgDir=/tmp/capi_cfg/proto_file_reader_creator_quicktest
inDir=/tmp/capi_in/proto_file_reader_creator_quicktest
outDir=/tmp/capi_out/proto_file_reader_creator_quicktest

if [ ! -d $cfgDir ]; then
  mkdir -p $cfgDir
else
  rm -f $cfgDir/*
fi

if [ ! -d $inDir ]; then
  mkdir -p $inDir
else
  rm -f $inDir/*
fi

if [ ! -d $outDir ]; then
  mkdir -p $outDir
else
  rm -f $outDir/*
fi

echo "Copying data files to "$inDir"..."
cp -r ../../data/in/proto_file_reader_creator_quicktest/* $inDir/

pushd ../../../pkg/exe/toolbelt
echo "Generating Capillaries script files to "$cfgDir"..."
go run capitoolbelt.go proto_file_reader_creator -file_type=csv -csv_hdr_line_idx=0 -csv_first_line_idx=1 -csv_separator="	" -file=$inDir/products.csv > $cfgDir/script_csv.json
go run capitoolbelt.go proto_file_reader_creator -file_type=parquet -file=$inDir/products.parquet > $cfgDir/script_parquet.json
popd

# Add "top" to the script so the output file is sorted and easy to compare to the baseline
sed -i -e 's~"url_template"~"top": {"order": "col_id(asc)"},\n    "url_template"~g' $cfgDir/script_csv.json
sed -i -e 's~"url_template"~"top": {"order": "col_id(asc)"},\n    "url_template"~g' $cfgDir/script_parquet.json
