#!/bin/bash

source ../common/util.sh

quick_or_big=$1
fs_or_s3=$2

if [[ "$quick_or_big" != "quick" && "$quick_or_big" != "big" || \
  "$fs_or_s3" != "fs" && "$fs_or_s3" != "s3" ]]; then
  echo $(basename "$0") requires 2 parameters:  'quick|big' 'fs|s3'
  exit 1
fi

dataDirName=lookup_${quick_or_big}test
cfgDir=/tmp/capi_cfg/$dataDirName
inDir=/tmp/capi_in/$dataDirName
outDir=/tmp/capi_out/$dataDirName

if [[ "$fs_or_s3" = "s3" ]]; then
  check_s3
  cfgS3=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_cfg/$dataDirName
  inS3=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_in/$dataDirName
fi

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

# Copy cfg files, generate in/out files

if [[ "$quick_or_big" = "big" ]]; then
  echo "Copying config files to "$cfgDir
  cp ../../data/cfg/lookup/script_big.json $cfgDir/
  cp ../../data/cfg/lookup/script_params_big_*.json $cfgDir/

  echo "Generating bigtest data from quicktest data..."
  # 100k items is a decent "quick" bigtest
  go run ./generate_data.go -formats=parquet -items=100000 -products=100 -sellers=200 -script_params_path=$cfgDir/script_params_big_fs_one.json -in_root=$inDir -out_root=$outDir -split_orders=10 -split_items=100
  if [ "$?" -ne "0" ]; then
    exit 1
  fi
else
  echo "Copying config files to "$cfgDir
  cp ../../data/cfg/lookup/script_quick.yaml $cfgDir/
  cp ../../data/cfg/lookup/script_params_quick_*.yaml $cfgDir/

  echo "Copying in files to "$inDir
  cp ../../data/in/lookup/* $inDir/

  echo "Copying out quicktest files to "$outDir
  echo "Placeholder for lookup_quicktest output files" > $outDir/readme.txt
  cp ../../data/out/lookup/* $outDir/
fi

if [[ "$fs_or_s3" == "s3" ]]; then
  echo "Patching cfg by specifying actual s3 bucket name..."
  if [[ "$quick_or_big" = "big" ]]; then
    sed -i -e 's~$CAPILLARIES_AWS_TESTBUCKET~'"$CAPILLARIES_AWS_TESTBUCKET"'~g' $cfgDir/script_params_big_s3_one.json
    sed -i -e 's~$CAPILLARIES_AWS_TESTBUCKET~'"$CAPILLARIES_AWS_TESTBUCKET"'~g' $cfgDir/script_params_big_s3_multi.json
  else
    sed -i -e 's~$CAPILLARIES_AWS_TESTBUCKET~'"$CAPILLARIES_AWS_TESTBUCKET"'~g' $cfgDir/script_params_quick_s3_one.yaml
    sed -i -e 's~$CAPILLARIES_AWS_TESTBUCKET~'"$CAPILLARIES_AWS_TESTBUCKET"'~g' $cfgDir/script_params_quick_s3_multi.yaml
  fi

  echo "Copying config files to "$cfgS3
  aws s3 rm $cfgS3/ --recursive
  aws s3 cp $cfgDir/ $cfgS3/ --recursive --include "*"

  echo "Copying in files to "$inS3
  aws s3 rm $inS3/ --recursive
  aws s3 cp $inDir/ $inS3/ --recursive --include "*"
fi