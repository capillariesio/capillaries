
cfgDir=/tmp/capi_cfg/lookup_quicktest
inDir=/tmp/capi_in/lookup_quicktest
outDir=/tmp/capi_out/lookup_quicktest

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

cp ../../../data/cfg/lookup_quicktest/* $cfgDir/

echo "Generating files..."

echo "Placeholder for lookup_quicktest output files" > $outDir/readme.txt

go run ../generate_data.go -items=1390 -products=10 -sellers=20 -script_params_path=$cfgDir/script_params_one_run.json -in_root=$inDir -out_root=$outDir
if [ "$?" -ne "0" ]; then
 exit 1
fi

