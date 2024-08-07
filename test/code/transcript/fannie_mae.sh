#!/bin/bash

source ./util.sh

OUT_FILE=transcript_fannie_mae.html
DATA_ROOT="../../data"
BUILD_DIR="../../../build/linux/amd64"
SCRIPT_JSON="$DATA_ROOT/cfg/fannie_mae_quicktest/script.json"

echo '<html><head><style>' > $OUT_FILE
cat transcript.css  >> $OUT_FILE
echo '</style></head><body>' >> $OUT_FILE
echo '<div class="container">' >> $OUT_FILE

echo "<div class="row"><h1>fannie_mae_quicktest script and data</h1></div>" >> $OUT_FILE

echo '<div class="row">' >> $OUT_FILE
echo "<h2>Input files</h2>" >> $OUT_FILE
echo "<h3>Payments</h3>" >> $OUT_FILE
$BUILD_DIR/capiparquet cat $DATA_ROOT/in/fannie_mae_quicktest/CAS_2023_R08_G1_20231020_000.parquet | python ./table_2_html.py >> $OUT_FILE
echo '</div>' >> $OUT_FILE

table2html payments 01_read_payments $SCRIPT_JSON $OUT_FILE fannie_mae_quicktest
table2html loan_ids 02_loan_ids $SCRIPT_JSON $OUT_FILE fannie_mae_quicktest
table2html deal_names 02_deal_names $SCRIPT_JSON $OUT_FILE fannie_mae_quicktest
table2html deal_total_upbs 03_deal_total_upbs $SCRIPT_JSON $OUT_FILE fannie_mae_quicktest
table2html loan_payment_summaries 04_loan_payment_summaries $SCRIPT_JSON $OUT_FILE fannie_mae_quicktest
table2html loan_summaries_calculated 04_loan_summaries_calculated $SCRIPT_JSON $OUT_FILE fannie_mae_quicktest
table2html deal_seller_summaries 05_deal_seller_summaries $SCRIPT_JSON $OUT_FILE fannie_mae_quicktest
table2html deal_summaries 05_deal_summaries $SCRIPT_JSON $OUT_FILE fannie_mae_quicktest

parquet2html "$DATA_ROOT/out/fannie_mae_quicktest/loan_summaries_calculated_baseline.parquet" 04_write_file_loan_summaries_calculated $SCRIPT_JSON $OUT_FILE
parquet2html "$DATA_ROOT/out/fannie_mae_quicktest/deal_seller_summaries_baseline.parquet" 05_write_file_deal_seller_summaries $SCRIPT_JSON $OUT_FILE
parquet2html "$DATA_ROOT/out/fannie_mae_quicktest/deal_summaries_baseline.parquet" 05_write_file_deal_summaries $SCRIPT_JSON $OUT_FILE

echo '</div></body></head>' >> $OUT_FILE
