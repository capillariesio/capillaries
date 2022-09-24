rm -f ./data/out/tag_totals.csv ./data/out/tagged_products_for_operator_review.csv ./data/out/runs.csv
pushd ../../pkg/exe/toolbelt
  go run toolbelt.go drop_keyspace -keyspace=test_tag_and_denormalize
popd