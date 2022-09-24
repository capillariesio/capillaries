echo "Generating files..."

go run generate_data.go -in_file=./data/in/raw -out_file_py=./data/out/raw_py -out_file_go=./data/out/raw_go -items=1100 -products=10 -sellers=20

# In

echo "Shuffling in files..."

head -n1 ./data/in/raw > ./data/in/header
tail -n+2 ./data/in/raw > ./data/in/data
shuf ./data/in/data -o ./data/in/data
split -d -nl/5 ./data/in/data ./data/in/data_chunk

cat ./data/in/header ./data/in/data_chunk00 > ./data/in/olist_order_items_dataset00.csv
cat ./data/in/header ./data/in/data_chunk01 > ./data/in/olist_order_items_dataset01.csv
cat ./data/in/header ./data/in/data_chunk02 > ./data/in/olist_order_items_dataset02.csv
cat ./data/in/header ./data/in/data_chunk03 > ./data/in/olist_order_items_dataset03.csv
cat ./data/in/header ./data/in/data_chunk04 > ./data/in/olist_order_items_dataset04.csv

rm ./data/in/raw ./data/in/header ./data/in/data*

# Out

echo "Shuffling out files..."

head -n1 ./data/out/raw_py > ./data/out/header
tail -n+2 ./data/out/raw_py > ./data/out/data
sort ./data/out/data -o ./data/out/data
cat ./data/out/header ./data/out/data > ./data/out/taxed_order_items_py_baseline.csv

rm ./data/out/raw_py ./data/out/header ./data/out/data

head -n1 ./data/out/raw_go > ./data/out/header
tail -n+2 ./data/out/raw_go > ./data/out/data
sort ./data/out/data -o ./data/out/data
cat ./data/out/header ./data/out/data > ./data/out/taxed_order_items_go_baseline.csv

rm ./data/out/raw_go ./data/out/header ./data/out/data
