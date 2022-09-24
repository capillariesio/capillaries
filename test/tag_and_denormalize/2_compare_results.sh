if ! diff -b ./data/out/tag_totals.tsv ./data/out/tag_totals_baseline.tsv; then
  echo "FAILED"
  exit 1
else
  echo "SUCCESS"
fi