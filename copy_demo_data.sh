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

cp -r ./test/data/cfg/* /tmp/capi_cfg
cp -r ./test/data/in/* /tmp/capi_in
cp -r ./test/data/out/* /tmp/capi_out
