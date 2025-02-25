#!/bin/bash

cfgDir=/tmp/capi_cfg/global_affairs_quicktest
inDir=/tmp/capi_in/global_affairs_quicktest
outDir=/tmp/capi_out/global_affairs_quicktest

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
cp -r ../../../data/cfg/global_affairs_quicktest/* $cfgDir/

echo "Copying in files to "$inDir
cp -r ../../../data/in/global_affairs_quicktest/* $inDir/

echo "Copying out files to "$outDir
cp -r ../../../data/out/global_affairs_quicktest/* $outDir/
