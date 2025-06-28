#!/bin/bash

source ../common/util.sh

quick_or_big=$1
fs_or_s3=$2

if [[ "$quick_or_big" != "quick" && "$quick_or_big" != "big" || "$fs_or_s3" != "fs" && "$fs_or_s3" != "s3" ]]; then
  echo $(basename "$0") requires 2 parameters:  'quick|big' 'fs|s3'
  exit 1
fi

if [[ "$quick_or_big" = "big" ]]; then
  dataDirName=portfolio_bigtest
else
  dataDirName=portfolio_quicktest
fi

if [[ "$fs_or_s3" = "s3" ]]; then
  check_s3
  cfgS3=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_cfg/$dataDirName
  inS3=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_in/$dataDirName
fi

cfgDir=/tmp/capi_cfg/$dataDirName
inDir=/tmp/capi_in/$dataDirName
outDir=/tmp/capi_out/$dataDirName


# Create /tmp dirs

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

# Copy cfg files, generate in/out files

echo "Copying Python files to "$cfgDir/py
cp ../../data/cfg/portfolio_quicktest/py/* $cfgDir/py/

if [[ "$quick_or_big" = "big" ]]; then
  echo "Copying config files to "$cfgDir
  cp ../../data/cfg/portfolio/script_big.json $cfgDir/
  cp ../../data/cfg/portfolio/script_params_big_*.json $cfgDir/

  echo "Generating bigtest data from quicktest data by cloning accounts and txns..."
  go run ./generate_bigtest_data.go -accounts=1000 -quicktest_in=../../data/in/portfolio -quicktest_out=../../data/out/portfolio -in_root=$inDir -out_root=$outDir

  echo "Sorting out files..."
  go run ../parquet/capiparquet.go sort $outDir/account_period_sector_perf_baseline.parquet 'ARK fund,Period,Sector'
  go run ../parquet/capiparquet.go sort $outDir/account_year_perf_baseline.parquet 'ARK fund,Period'
else
  echo "Copying config files to "$cfgDir
  cp ../../data/cfg/portfolio/script_quick.json $cfgDir/
  cp ../../data/cfg/portfolio/script_params_quick_*.json $cfgDir/

  echo "Copying in files to "$inDir
  cp ../../data/in/portfolio/* $inDir/

  echo "Copying out quicktest files to "$outDir
  echo "Placeholder for portfolio_quicktest output files" > $outDir/readme.txt
  cp ../../data/out/portfolio/* $outDir/
fi

if [[ "$fs_or_s3" == "s3" ]]; then
  echo "Patching cfg by specifying actual s3 backet name..."
  sed -i -e 's~$CAPILLARIES_AWS_TESTBUCKET~'"$CAPILLARIES_AWS_TESTBUCKET"'~g' $cfgDir/script_params_${quick_or_big}_s3_one.json
  sed -i -e 's~$CAPILLARIES_AWS_TESTBUCKET~'"$CAPILLARIES_AWS_TESTBUCKET"'~g' $cfgDir/script_params_${quick_or_big}_s3_multi.json

  echo "Copying config files to "$cfgS3
  aws s3 rm $cfgS3/ --recursive
  aws s3 cp $cfgDir/ $cfgS3/ --recursive --include "*"
  aws s3 cp $cfgDir/py/ $cfgS3/py/ --recursive --include "*"

  echo "Copying in files to "$inS3
  aws s3 rm $inS3/ --recursive
  aws s3 cp $inDir/ $inS3/ --recursive --include "*"
fi