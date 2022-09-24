rm -f ./data/in/raw ./data/in/header ./data/in/data* ./data/in/*.csv
rm -f ./data/out/raw ./data/out/header ./data/out/data* ./data/out/*.csv
pushd ../../pkg/exe/toolbelt
  go run toolbelt.go drop_keyspace -keyspace=test_py_calc
popd