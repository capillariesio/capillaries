inDir=/tmp/capitest_in/lookup
outDir=/tmp/capitest_out/lookup

echo "Generating files..."

go run generate_data.go -items=1390 -products=10 -sellers=20

# In

echo "Shuffling in files..."

# Orders
head -n1 $inDir/raw_orders > $inDir/header
tail -n+2 $inDir/raw_orders > $inDir/data
shuf $inDir/data -o $inDir/data
cat $inDir/header $inDir/data > $inDir/olist_orders_dataset.csv

# Items
head -n1 $inDir/raw_items > $inDir/header
tail -n+2 $inDir/raw_items > $inDir/data
shuf $inDir/data -o $inDir/data
cat $inDir/header $inDir/data > $inDir/olist_order_items_dataset.csv

rm $inDir/raw* $inDir/header $inDir/data*

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
