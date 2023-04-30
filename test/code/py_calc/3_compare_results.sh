#!/bin/bash

outDir=/tmp/capi_out/py_calc_quicktest

echo "Merging out files..."

# Taxed (python)
head -n1 $outDir/taxed_order_items_py_00000.csv > $outDir/header

tail -n+2 $outDir/taxed_order_items_py_00000.csv > $outDir/data0
tail -n+2 $outDir/taxed_order_items_py_00001.csv > $outDir/data1

cat $outDir/data0 $outDir/data1 > $outDir/data

sort $outDir/data -o $outDir/data

cat $outDir/header $outDir/data > $outDir/taxed_order_items_py.csv

# Simple taxed (table-to-table)
head -n1 $outDir/taxed_order_items_go_00000.csv > $outDir/header

tail -n+2 $outDir/taxed_order_items_go_00000.csv > $outDir/data0
tail -n+2 $outDir/taxed_order_items_go_00001.csv > $outDir/data1

cat $outDir/data0 $outDir/data1 > $outDir/data

sort $outDir/data -o $outDir/data

cat $outDir/header $outDir/data > $outDir/taxed_order_items_go.csv

# Clean up

rm $outDir/header $outDir/data $outDir/data0 $outDir/data1


if ! diff -b $outDir/taxed_order_items_py_baseline.csv $outDir/taxed_order_items_py.csv; then
  echo "Python custom processor test FAILED"
  exit 1
else
  echo "Python custom processor test SUCCESS"
fi

if ! diff -b $outDir/taxed_order_items_go_baseline.csv $outDir/taxed_order_items_go.csv; then
  echo "Golang table-to-table test FAILED"
  exit 1
else
  echo "Golang table-to-table test SUCCESS"
fi