
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

cp ../../data/cfg/lookup_quicktest/* $cfgDir/

echo "Generating files..."

echo "Placeholder for lookup_quicktest output files" > $outDir/readme.txt

go run generate_data.go -items=1390 -products=10 -sellers=20 -script_params_path=$cfgDir/script_params_one_run.json -in_root=$inDir -out_root=$outDir
if [ "$?" -ne "0" ]; then
 exit 1
fi

# Orders

# head -n1 $inDir/raw_orders > $inDir/header
# tail -n+2 $inDir/raw_orders > $inDir/data

# echo "Shuffling orders..."
# shuf $inDir/data -o $inDir/data

# echo "Finalizing order file..."
# cat $inDir/header $inDir/data > $inDir/olist_orders_dataset.csv

# Items

# head -n1 $inDir/raw_items > $inDir/header
# tail -n+2 $inDir/raw_items > $inDir/data

# echo "Shuffling order items..."
# shuf $inDir/data -o $inDir/data

# echo "Finalizing order items files..."
# cat $inDir/header $inDir/data > $inDir/olist_order_items_dataset.csv

# rm $inDir/raw* $inDir/header $inDir/data*

# Out

# echo "Sorting baseline files..."

# head -n1 $outDir/raw_no_group_inner > $outDir/header
# tail -n+2 $outDir/raw_no_group_inner > $outDir/data
# sort $outDir/data -o $outDir/data
# cat $outDir/header $outDir/data > $outDir/order_item_date_inner_baseline.csv

# head -n1 $outDir/raw_no_group_outer > $outDir/header
# tail -n+2 $outDir/raw_no_group_outer > $outDir/data
# sort $outDir/data -o $outDir/data
# cat $outDir/header $outDir/data > $outDir/order_item_date_left_outer_baseline.csv

# head -n1 $outDir/raw_grouped_inner > $outDir/header
# tail -n+2 $outDir/raw_grouped_inner > $outDir/data
# sort -r $outDir/data -o $outDir/data
# cat $outDir/header $outDir/data > $outDir/order_date_value_grouped_inner_baseline.csv

# head -n1 $outDir/raw_grouped_outer > $outDir/header
# tail -n+2 $outDir/raw_grouped_outer > $outDir/data
# sort -r $outDir/data -o $outDir/data
# cat $outDir/header $outDir/data > $outDir/order_date_value_grouped_left_outer_baseline.csv

# rm $outDir/raw* $outDir/header $outDir/data*
