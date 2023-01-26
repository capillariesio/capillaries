
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

cp ../../data/cfg/lookup_bigtest/* $cfgDir/

echo "Generating files..."

echo "Placeholder for lookup_bigtest output files" > $outDir/readme.txt

go run generate_data.go -items=10000 -products=100 -sellers=200 -script_params_path=$cfgDir/script_params_one_run.json -in_root=$inDir -out_root=$outDir
if [ "$?" -ne "0" ]; then
 exit 1
fi

# Orders

head -n1 $inDir/raw_orders > $inDir/header
tail -n+2 $inDir/raw_orders > $inDir/data

echo "Shuffling orders..."
shuf $inDir/data -o $inDir/data

echo "Splitting order file..."
split -d -nl/10 $inDir/data $inDir/data_chunk

echo "Finalizing order files..."
for i in $(seq -f "%02g" 00 09)
do
  cat $inDir/header $inDir/data_chunk$i > $inDir/olist_orders_dataset$i.csv
done

# Items

head -n1 $inDir/raw_items > $inDir/header
tail -n+2 $inDir/raw_items > $inDir/data

echo "Shuffling order items..."
shuf $inDir/data -o $inDir/data

echo "Splitting order items file..."
split -d -nl/100 $inDir/data $inDir/data_chunk

echo "Finalizing order items files..."
for i in $(seq -f "%02g" 00 99)
do
  cat $inDir/header $inDir/data_chunk$i > $inDir/olist_orders_item_dataset$i.csv
done

rm $inDir/raw* $inDir/header $inDir/data*

echo "Packing input files..."

pushd $inDir
tar -czf $inDir/all.tgz .
popd

# Out

echo "Sorting baseline files..."

head -n1 $outDir/raw_no_group_inner > $outDir/header
tail -n+2 $outDir/raw_no_group_inner > $outDir/data
sort $outDir/data -o $outDir/data
cat $outDir/header $outDir/data > $outDir/order_item_date_inner_baseline.csv

head -n1 $outDir/raw_no_group_outer > $outDir/header
tail -n+2 $outDir/raw_no_group_outer > $outDir/data
sort $outDir/data -o $outDir/data
cat $outDir/header $outDir/data > $outDir/order_item_date_left_outer_baseline.csv

head -n1 $outDir/raw_grouped_inner > $outDir/header
tail -n+2 $outDir/raw_grouped_inner > $outDir/data
sort -r $outDir/data -o $outDir/data
cat $outDir/header $outDir/data > $outDir/order_date_value_grouped_inner_baseline.csv

head -n1 $outDir/raw_grouped_outer > $outDir/header
tail -n+2 $outDir/raw_grouped_outer > $outDir/data
sort -r $outDir/data -o $outDir/data
cat $outDir/header $outDir/data > $outDir/order_date_value_grouped_left_outer_baseline.csv

rm $outDir/raw* $outDir/header $outDir/data*


