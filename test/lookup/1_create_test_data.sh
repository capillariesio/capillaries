echo "Generating files..."

go run generate_data.go -items=13900 -products=10 -sellers=20

# In

echo "Shuffling in files..."

# Orders
head -n1 ./data/in/raw_orders > ./data/in/header
tail -n+2 ./data/in/raw_orders > ./data/in/data
shuf ./data/in/data -o ./data/in/data
cat ./data/in/header ./data/in/data > ./data/in/olist_orders_dataset.csv

# Items
head -n1 ./data/in/raw_items > ./data/in/header
tail -n+2 ./data/in/raw_items > ./data/in/data
shuf ./data/in/data -o ./data/in/data
cat ./data/in/header ./data/in/data > ./data/in/olist_order_items_dataset.csv

rm ./data/in/raw* ./data/in/header ./data/in/data*

# Out

echo "Shuffling out files..."

head -n1 ./data/out/raw_no_group_inner > ./data/out/header
tail -n+2 ./data/out/raw_no_group_inner > ./data/out/data
sort ./data/out/data -o ./data/out/data
cat ./data/out/header ./data/out/data > ./data/out/order_item_date_inner_baseline.csv

head -n1 ./data/out/raw_no_group_outer > ./data/out/header
tail -n+2 ./data/out/raw_no_group_outer > ./data/out/data
sort ./data/out/data -o ./data/out/data
cat ./data/out/header ./data/out/data > ./data/out/order_item_date_left_outer_baseline.csv

head -n1 ./data/out/raw_grouped_inner > ./data/out/header
tail -n+2 ./data/out/raw_grouped_inner > ./data/out/data
sort -r ./data/out/data -o ./data/out/data
cat ./data/out/header ./data/out/data > ./data/out/order_date_value_grouped_inner_baseline.csv

head -n1 ./data/out/raw_grouped_outer > ./data/out/header
tail -n+2 ./data/out/raw_grouped_outer > ./data/out/data
sort -r ./data/out/data -o ./data/out/data
cat ./data/out/header ./data/out/data > ./data/out/order_date_value_grouped_left_outer_baseline.csv

rm ./data/out/raw* ./data/out/header ./data/out/data*
