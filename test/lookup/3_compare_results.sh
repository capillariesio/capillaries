if ! diff -b ./data/out/order_item_date_inner_baseline.csv ./data/out/order_item_date_inner.csv ||
  ! diff -b ./data/out/order_item_date_left_outer_baseline.csv ./data/out/order_item_date_left_outer.csv ||
  ! diff -b ./data/out/order_date_value_grouped_inner_baseline.csv ./data/out/order_date_value_grouped_inner.csv ||
  ! diff -b ./data/out/order_date_value_grouped_left_outer_baseline.csv ./data/out/order_date_value_grouped_left_outer.csv; then
  echo "FAILED"
  exit 1
else
  echo "SUCCESS"
fi