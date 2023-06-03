cfgDir=/tmp/capi_cfg/lookup_bigtest
inDir=/tmp/capi_in/lookup_bigtest
outDir=/tmp/capi_out/lookup_bigtest

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

cp ../../../data/cfg/lookup_bigtest/* $cfgDir/

echo "Generating files..."

echo "Placeholder for lookup_bigtest output files" > $outDir/readme.txt

go run ../generate_data.go -formats=csv,parquet -items=100000 -products=100 -sellers=200 -script_params_path=$cfgDir/script_params_one_run.json -in_root=$inDir -out_root=$outDir -split_orders=10 -split_items=100
if [ "$?" -ne "0" ]; then
 exit 1
fi


echo "Packing input and ouput files..."

pushd $inDir
tar -czf $inDir/all_csv.tgz *.csv
tar -czf $inDir/all_parquet.tgz *.parquet
popd

pushd $outDir
tar -czf $outDir/all_csv.tgz *.csv
tar -czf $outDir/all_parquet.tgz *.parquet
popd