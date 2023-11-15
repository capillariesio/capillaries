#!/bin/bash

outDir=/mnt/capi_out/tag_and_denormalize_quicktest

sort $outDir/tagged_products_for_operator_review.csv -o $outDir/tagged_products_for_operator_review_sorted.csv

if ! diff -b $outDir/tag_totals.tsv $outDir/tag_totals_baseline.tsv ||
  ! diff -b $outDir/tagged_products_for_operator_review_sorted.csv $outDir/tagged_products_for_operator_review_baseline.csv; then
  echo -e "tag_and_denormalize_quicktest diff \033[0;31mFAILED\e[0m"
  exit 1
else
  echo -e "tag_and_denormalize_quicktest diff \033[0;32mOK\e[0m"
fi