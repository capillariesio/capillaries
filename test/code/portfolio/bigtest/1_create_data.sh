#!/bin/bash

source ../../common/util.sh
check_s3

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

echo "Generating bigtest data from quicktest data by cloning accounts and txns..."
go run ./generate_bigtest_data.go -accounts=1000 -quicktest_in=../../../data/in/portfolio_quicktest quicktest_out=../../../data/out/portfolio_quicktest -in_root=$inDir -out_root=$outDir

echo "Sorting out files..."
go run ../../parquet/capiparquet.go sort $outDir/account_period_sector_perf_baseline.parquet 'ARK fund,Period,Sector'
go run ../../parquet/capiparquet.go sort $outDir/account_year_perf_baseline.parquet 'ARK fund,Period'

echo "Patching cfg..."
sed -i -e 's~$CAPILLARIES_AWS_TESTBUCKET~'"$CAPILLARIES_AWS_TESTBUCKET"'~g' $cfgDir/script_params.json

cfgS3=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_cfg/portfolio_bigtest
inS3=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_in/portfolio_bigtest

echo "Copying config files to "$cfgS3
aws s3 cp $cfgDir/ $cfgS3/ --recursive --include "*"

echo "Copying in files to "$inS3
aws s3 cp $inDir/ $inS3/ --recursive --include "*"
