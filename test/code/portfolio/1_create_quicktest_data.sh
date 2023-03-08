cfgDir=/tmp/capi_cfg/portfolio_quicktest
inDir=/tmp/capi_in/portfolio_quicktest
outDir=/tmp/capi_out/portfolio_quicktest

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
cp -r ../../data/cfg/portfolio_quicktest/* $cfgDir/

echo "Copying in files to "$inDir
cp -r ../../data/in/portfolio/* $inDir/

echo "Copying out files to "$outDir
echo "Placeholder for portfolio_quicktest output files" > $outDir/readme.txt
cp -r ../../data/out/portfolio/* $outDir/
