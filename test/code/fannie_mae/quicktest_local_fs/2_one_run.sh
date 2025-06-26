#!/bin/bash

source ../../common/util.sh

keyspace="fannie_mae_quicktest"
scriptFile=/tmp/capi_cfg/fannie_mae_quicktest/script.json
paramsFile=/tmp/capi_cfg/fannie_mae_quicktest/script_params.json
outDir=/tmp/capi_out/fannie_mae_quicktest

one_daemon_run  $keyspace $scriptFile $paramsFile $outDir '01_read_payments'

02_loan_ids

02_deal_names,02_deal_selers,04_loan_payment_summaries

05_deal_summaries,05_deal_seller_summaries