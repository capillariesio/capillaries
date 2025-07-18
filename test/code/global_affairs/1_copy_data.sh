#!/bin/bash

source ../common/util.sh

quick_or_big=$1
fs_or_s3=$2

if [[ "$quick_or_big" != "quick" && "$quick_or_big" != "big" || \
  "$fs_or_s3" != "fs" && "$fs_or_s3" != "s3" ]]; then
  echo $(basename "$0") requires 2 parameters:  'quick|big' 'fs|s3'
  exit 1
fi

dataDirName=global_affairs_${quick_or_big}test
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

echo "Copying config files to "$cfgDir
cp -r ../../data/cfg/global_affairs/* $cfgDir/

echo "Copying in files to "$inDir
cp -r ../../data/in/global_affairs/countries.csv $inDir/
cp -r ../../data/in/global_affairs/harvested_projects.csv $inDir/
cp -r ../../data/in/global_affairs/partners.csv $inDir/
cp -r ../../data/in/global_affairs/projects.csv $inDir/
cp -r ../../data/in/global_affairs/sectors.csv $inDir/
if [[ "$quick_or_big" = "big" ]]; then
  cp -r ../../data/in/global_affairs/harvested_project_financials_bigtest.csv $inDir/
else
  cp -r ../../data/in/global_affairs/harvested_project_financials_quicktest.csv $inDir/
fi

if [[ "$quick_or_big" = "quick" ]]; then
  echo "Copying out files to "$outDir
  cp -r ../../data/out/global_affairs/* $outDir/
fi

if [[ "$fs_or_s3" = "s3" ]]; then
  echo "Patching cfg by specifying actual s3 bucket name..."
  sed -i -e 's~$CAPILLARIES_AWS_TESTBUCKET~'"$CAPILLARIES_AWS_TESTBUCKET"'~g' $cfgDir/script_params_${quick_or_big}_s3_one.json
  sed -i -e 's~$CAPILLARIES_AWS_TESTBUCKET~'"$CAPILLARIES_AWS_TESTBUCKET"'~g' $cfgDir/script_params_${quick_or_big}_s3_multi.json

  echo "Copying config files to "$cfgS3
  aws s3 rm $cfgS3/ --recursive
  aws s3 cp $cfgDir/ $cfgS3/ --recursive --include "*"

  echo "Copying in files to "$inS3
  aws s3 rm $inS3/ --recursive
  aws s3 cp $inDir/ $inS3/ --recursive --include "*"
fi