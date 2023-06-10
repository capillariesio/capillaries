rm -fR /tmp/capi_cfg
rm -fR /tmp/capi_in
rm -fR /tmp/capi_out

mkdir /tmp/capi_cfg
mkdir /tmp/capi_in
mkdir /tmp/capi_out

cp -r ./test/data/cfg/* /tmp/capi_cfg
cp -r ./test/data/in/* /tmp/capi_in
cp -r ./test/data/out/* /tmp/capi_out

