#!/bin/bash

cfgDir=/tmp/capi_cfg/portfolio_bigtest
inDir=/tmp/capi_in/portfolio_bigtest
outDir=/tmp/capi_out/portfolio_bigtest

if [ ! -d $cfgDir ]; then
  mkdir -p $cfgDir
else
  rm -fR $cfgDir/*
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

if [ ! -d $cfgDir/py ]; then
  mkdir -p $cfgDir/py
else
  rm -f $cfgDir/py/*
fi

echo "Copying config files to "$cfgDir
cp ../../../data/cfg/portfolio_bigtest/* $cfgDir/

echo "Copying Python files to "$cfgDir/py
cp -r ../../../data/cfg/portfolio_quicktest/py/* $cfgDir/py/

echo "Generating data..."
go run ./generate_bigtest_data.go -accounts=1000

echo "Sorting out files..."
go run ../../parquet/capiparquet.go sort $outDir/account_period_sector_perf_baseline.parquet 'ARK fund,Period,Sector'
go run ../../parquet/capiparquet.go sort $outDir/account_year_perf_baseline.parquet 'ARK fund,Period'


echo "Packing input and ouput files..."

pushd $inDir
tar -czf $inDir/all.tgz *.parquet
popd

pushd $outDir
tar -czf $outDir/all.tgz *.parquet
popd