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
  echo -e "Python custom processor test \033[0;31mFAILED\e[0m"
  exit 1
else
  echo -e "Python custom processor test \033[0;32mOK\e[0m"
fi

if ! diff -b $outDir/taxed_order_items_go_baseline.csv $outDir/taxed_order_items_go.csv; then
  echo -e "Golang table-to-table test \033[0;31mFAILED\e[0m"
  exit 1
else
  echo -e "Golang table-to-table test \033[0;32mOK\e[0m"
fi