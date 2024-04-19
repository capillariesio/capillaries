#!/bin/bash

source ../../common/util.sh
check_s3

srcDir=../../../data
cfgSrcDir=$srcDir/cfg/fannie_mae_quicktest
inSrcDir=$srcDir/in/fannie_mae_quicktest
outSrcDir=$srcDir/out/fannie_mae_quicktest

cfgDir=/tmp/capi_cfg/fannie_mae_quicktest
outDir=/tmp/capi_out/fannie_mae_quicktest

cfgS3=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_cfg/fannie_mae_quicktest
inS3=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_in/fannie_mae_quicktest

if [ ! -d $cfgDir ]; then
  mkdir -p $cfgDir
else
  rm -fR $cfgDir/*
fi

if [ ! -d $outDir ]; then
  mkdir -p $outDir
else
  rm -fR $outDir/*
fi

echo "Copying config files to "$cfgS3
sed -e 's~$CAPILLARIES_AWS_TESTBUCKET~'$CAPILLARIES_AWS_TESTBUCKET'~g' $cfgSrcDir/script_params_s3.json > $cfgDir/script_params_s3.json
aws s3 cp $cfgDir/script_params_s3.json $cfgS3/
aws s3 cp $cfgSrcDir/py $cfgS3/py --recursive
aws s3 cp $cfgSrcDir/script.json $cfgS3/

echo "Copying in files to "$inS3
aws s3 cp $inSrcDir $inS3/ --recursive --exclude "*" --include "CAS_2023_R08_G1_20231020_*.parquet"

echo "Copying files to "$outDir
cp -r $outSrcDir/* $outDir/
