source ../../common/util.sh
check_s3

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

# 100k items is a decent "quick" bigtest
go run ../generate_data.go -formats=parquet,csv -items=100000 -products=100 -sellers=200 -script_params_path=$cfgDir/script_params_one_run.json -in_root=$inDir -out_root=$outDir -split_orders=10 -split_items=100
if [ "$?" -ne "0" ]; then
 exit 1
fi

echo "Patching cfg..."
sed -i -e 's~$CAPILLARIES_AWS_TESTBUCKET~'"$CAPILLARIES_AWS_TESTBUCKET"'~g' $cfgDir/script_params_one_run.json

cfgS3=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_cfg/lookup_bigtest
inS3=s3://$CAPILLARIES_AWS_TESTBUCKET/capi_in/lookup_bigtest

echo "Copying config files to "$cfgS3
aws s3 cp $cfgDir/ $cfgS3/ --recursive --include "*"

echo "Copying in files to "$inS3
aws s3 cp $inDir/ $inS3/ --recursive --include "*"
