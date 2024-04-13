if [ ! -e ~/.aws/credentials ]; then
  echo ~/.aws/credentials not found
  exit 1
fi

if [ ! -e ~/.aws/config ]; then
  echo ~/.aws/config not found
  exit 1
fi

./1_create_data.sh

# Copy to s3

cfgDir=/tmp/capi_cfg/lookup_quicktest
inDir=/tmp/capi_in/lookup_quicktest

s3Url=s3://capillaries-sampledeployment005
s3UrlIn=$s3Url/capi_in/lookup_quicktest
s3UrlCfg=$s3Url/capi_cfg/lookup_quicktest

aws s3 cp $cfgDir/script.json $s3UrlCfg/
aws s3 cp $cfgDir/script_params_one_run_s3.json $s3UrlCfg/
aws s3 cp $inDir/olist_order_items_dataset.csv $s3UrlIn/
aws s3 cp $inDir/olist_orders_dataset.csv $s3UrlIn/


