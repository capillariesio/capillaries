#!/bin/bash

outDir="../../data/out/tag_and_denormalize"

if ! diff -b $outDir/tag_totals.tsv $outDir/tag_totals_baseline.tsv; then
  echo "FAILED"
  exit 1
else
  echo "SUCCESS"
fi