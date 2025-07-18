sudo rm -fR /tmp/capi_cfg
sudo rm -fR /tmp/capi_in
sudo rm -fR /tmp/capi_out
sudo rm -fR /tmp/capi_log

mkdir /tmp/capi_cfg
mkdir /tmp/capi_in
mkdir /tmp/capi_out
mkdir /tmp/capi_log

chmod 777 /tmp/capi_cfg
chmod 777 /tmp/capi_in
chmod 777 /tmp/capi_out
chmod 777 /tmp/capi_log

pushd test/code/lookup
./1_create_data.sh quick fs
popd

pushd test/code/py_calc
./1_copy_data.sh
popd

pushd test/code/tag_and_denormalize
./1_copy_data.sh
popd

pushd test/code/portfolio
./1_create_data.sh quick fs
popd

