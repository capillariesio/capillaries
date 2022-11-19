#!/bin/bash

inDir=/tmp/capitest_in/py_calc
outDir=/tmp/capitest_out/py_calc

echo "Generating files..."

go run generate_data.go -in_file=$inDir/raw -out_file_py=$outDir/raw_py -out_file_go=$outDir/raw_go -items=1100 -products=10 -sellers=20

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
