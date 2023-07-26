cfgDir=/tmp/capi_cfg/portfolio_bigtest
inDir=/tmp/capi_in/portfolio_bigtest
outDir=/tmp/capi_out/portfolio_bigtest

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

echo "Copying Python files to "$cfgDir/py
cp -r ../../../data/cfg/portfolio_quicktest/py/* $cfgDir/py/

echo "Generating data.."
go run ./generate_bigtest_data.go -accounts=100
