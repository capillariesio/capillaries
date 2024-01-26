#!/bin/bash

cfgDir=/tmp/capi_cfg/fannie_mae_bigtest
inDir=/tmp/capi_in/fannie_mae_bigtest
outDir=/tmp/capi_out/fannie_mae_bigtest

if [ ! -d $cfgDir ]; then
  mkdir -p $cfgDir
else
  rm -fR $cfgDir/*
fi

if [ ! -d $cfgDir/py ]; then
  mkdir -p $cfgDir/py
else
  rm -fR $cfgDir/py/*
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
cp ../../../data/cfg/fannie_mae_bigtest/* $cfgDir/
cp ../../../data/cfg/fannie_mae_quicktest/py/* $cfgDir/py/

echo "Copying in files to "$inDir
cp -r ../../../../../capillaries-fanniemae/parquet/CAS_2023_*.parquet $inDir/

echo "Copying out files to "$outDir
echo "Placeholder for fannie_mae_bigtest output files" > $outDir/readme.txt
cp -r ../../../data/out/fannie_mae_bigtest/* $outDir/
