#!/bin/bash

cfgDir=/tmp/capi_cfg/fannie_mae_quicktest
inDir=/tmp/capi_in/fannie_mae_quicktest
outDir=/tmp/capi_out/fannie_mae_quicktest

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
cp -r ../../../data/cfg/fannie_mae_quicktest/* $cfgDir/
echo "Copying in files to "$inDir
cp -r ../../../data/in/fannie_mae_quicktest/* $inDir/
echo "Copying out files to "$outDir
echo "Placeholder for fannie_mae_quicktest output files" > $outDir/readme.txt
cp -r ../../../data/out/fannie_mae_quicktest/* $outDir/
