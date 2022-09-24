./4_clean.sh
./1_create_test_data.sh
./2_two_runs.sh
if ! ./3_compare_results.sh; then
  echo "NOT CLEANED"
  exit 1
else
  ./4_clean.sh
fi
