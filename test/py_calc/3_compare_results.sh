echo "Merging out files..."

# Taxed (python)
head -n1 ./data/out/taxed_order_items_py_00000.csv > ./data/out/header

tail -n+2 ./data/out/taxed_order_items_py_00000.csv > ./data/out/data0
tail -n+2 ./data/out/taxed_order_items_py_00001.csv > ./data/out/data1

cat ./data/out/data0 ./data/out/data1 > ./data/out/data

sort ./data/out/data -o ./data/out/data

cat ./data/out/header ./data/out/data > ./data/out/taxed_order_items_py.csv

# Simple taxed (table-to-table)
head -n1 ./data/out/taxed_order_items_go_00000.csv > ./data/out/header

tail -n+2 ./data/out/taxed_order_items_go_00000.csv > ./data/out/data0
tail -n+2 ./data/out/taxed_order_items_go_00001.csv > ./data/out/data1

cat ./data/out/data0 ./data/out/data1 > ./data/out/data

sort ./data/out/data -o ./data/out/data

cat ./data/out/header ./data/out/data > ./data/out/taxed_order_items_go.csv

# Clean up

rm ./data/out/header ./data/out/data ./data/out/data0 ./data/out/data1


if ! diff -b ./data/out/taxed_order_items_py_baseline.csv ./data/out/taxed_order_items_py.csv; then
  echo "Python custom processor test FAILED"
  exit 1
else
  echo "Python custom processor test SUCCESS"
fi

if ! diff -b ./data/out/taxed_order_items_go_baseline.csv ./data/out/taxed_order_items_go.csv; then
  echo "Golang table-to-table test FAILED"
  exit 1
else
  echo "Golang table-to-table test SUCCESS"
fi