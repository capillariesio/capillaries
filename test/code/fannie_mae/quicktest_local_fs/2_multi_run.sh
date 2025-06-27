#!/bin/bash

source ../../common/util.sh

keyspace="fannie_mae_quicktest"
scriptFile=/tmp/capi_cfg/fannie_mae_quicktest/script.json
paramsFile=/tmp/capi_cfg/fannie_mae_quicktest/script_params.json
startNodes1='01_read_payments'
startNodes2='02_loan_ids'
startNodes3='02_deal_names,02_deal_sellers,04_loan_payment_summaries'
startNodes4='03_deal_total_upbs,04_loan_smrs_clcltd'
startNodes5='05_deal_summaries,05_deal_seller_summaries'

webapi_multi_run 'http://localhost:6543' $keyspace $scriptFile $paramsFile $startNodes1 $startNodes2 $startNodes3 $startNodes4 $startNodes5