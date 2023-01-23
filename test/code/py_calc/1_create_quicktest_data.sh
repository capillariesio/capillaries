#!/bin/bash

cfgDir=/tmp/capi_cfg/py_calc_quicktest
inDir=/tmp/capi_in/py_calc_quicktest
outDir=/tmp/capi_out/py_calc_quicktest

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

echo "Copying config files to "$cfgDir

cp -r ../../data/cfg/py_calc_quicktest/* $cfgDir/

echo "Generating files..."

go run generate_data.go -in_file=$inDir/raw -out_file_py=$outDir/raw_py -out_file_go=$outDir/raw_go -items=1100 -products=10 -sellers=20
if [ "$?" -ne "0" ]; then
  exit
fi

# In

echo "Shuffling in files..."

head -n1 $inDir/raw > $inDir/header
tail -n+2 $inDir/raw > $inDir/data
shuf $inDir/data -o $inDir/data
split -d -nl/5 $inDir/data $inDir/data_chunk

cat $inDir/header $inDir/data_chunk00 > $inDir/olist_order_items_dataset00.csv
cat $inDir/header $inDir/data_chunk01 > $inDir/olist_order_items_dataset01.csv
cat $inDir/header $inDir/data_chunk02 > $inDir/olist_order_items_dataset02.csv
cat $inDir/header $inDir/data_chunk03 > $inDir/olist_order_items_dataset03.csv
cat $inDir/header $inDir/data_chunk04 > $inDir/olist_order_items_dataset04.csv

rm $inDir/raw $inDir/header $inDir/data*

# Out

echo "Sorting baseline files..."

head -n1 $outDir/raw_py > $outDir/header
tail -n+2 $outDir/raw_py > $outDir/data
sort $outDir/data -o $outDir/data
cat $outDir/header $outDir/data > $outDir/taxed_order_items_py_baseline.csv

rm $outDir/raw_py $outDir/header $outDir/data

head -n1 $outDir/raw_go > $outDir/header
tail -n+2 $outDir/raw_go > $outDir/data
sort $outDir/data -o $outDir/data
cat $outDir/header $outDir/data > $outDir/taxed_order_items_go_baseline.csv

rm $outDir/raw_go $outDir/header $outDir/data
