echo "Make sure that pkg/exe/toolbelt has access to Cassandra and RabbitMQ"

./3_clean.sh
./1_one_run.sh
if ! ./2_compare_results.sh; then
  echo "NOT CLEANED"
  exit 1
else
  ./3_clean.sh
fi
