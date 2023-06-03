for fname in $(find ../../data/parquet -type f -name '*.parquet'); do
  
  if ! go run caparquet.go cat $fname !>/dev/null; then
    echo -e $fname "\033[0;31mFAILED\e[0m"
  else
    echo -e $fname "\033[0;32mOK\e[0m"
  fi
done