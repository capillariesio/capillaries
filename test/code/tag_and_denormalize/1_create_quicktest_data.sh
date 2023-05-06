cfgDir=/tmp/capi_cfg/tag_and_denormalize_quicktest
inDir=/tmp/capi_in/tag_and_denormalize_quicktest
outDir=/tmp/capi_out/tag_and_denormalize_quicktest

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
cp -r ../../data/cfg/tag_and_denormalize_quicktest/* $cfgDir/
echo "Copying in files to "$inDir
cp -r ../../data/in/tag_and_denormalize_quicktest/* $inDir/
echo "Copying out files to "$outDir
echo "Placeholder for tag_and_denormalize_quicktest output files" > $outDir/readme.txt
cp -r ../../data/out/tag_and_denormalize_quicktest/* $outDir/
