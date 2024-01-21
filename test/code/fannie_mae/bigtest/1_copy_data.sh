#!/bin/bash

cfgDir=/tmp/capi_cfg/fannie_mae_bigtest
inDir=/tmp/capi_in/fannie_mae_bigtest
outDir=/tmp/capi_out/fannie_mae_bigtest

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
cp -r ../../../data/cfg/fannie_mae_bigtest/* $cfgDir/
echo "Copying in files to "$inDir
cp -r ../../../../../capillaries-fanniemae/parquet/CAS_2023_*.parquet $inDir/
echo "Copying out files to "$outDir
echo "Placeholder for fannie_mae_bigtest output files" > $outDir/readme.txt
cp -r ../../../data/out/fannie_mae_bigtest/* $outDir/
