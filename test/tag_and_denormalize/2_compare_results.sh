if ! diff -b ./data/out/tag_totals.csv ./data/out/tag_totals_baseline.csv; then
  echo "FAILED"
  exit 1
else
  echo "SUCCESS"
fi