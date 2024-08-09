#!/bin/bash

source ./util.sh

OUT_FILE=transcript_portfolio.md
DATA_ROOT="../../data"
SCRIPT_JSON="$DATA_ROOT/cfg/portfolio_quicktest/script.json"

if [[ $OUT_FILE == *html ]]; then
	echo '<html><head><style>' > $OUT_FILE
	cat transcript.css  >> $OUT_FILE
	echo '</style></head><body>' >> $OUT_FILE
	echo '<div class="container">' >> $OUT_FILE

	echo "<div class="row"><h1>portfolio_quicktest script and data</h1></div>" >> $OUT_FILE

	echo '<div class="row">' >> $OUT_FILE
	echo "<h2>Input files</h2>" >> $OUT_FILE
	echo "<h3>Accounts</h3>" >> $OUT_FILE
	cat $DATA_ROOT/in/portfolio_quicktest/accounts.csv | python ./table_2_html.py >> $OUT_FILE
	echo "<h3>Transactions</h3>" >> $OUT_FILE
	cat $DATA_ROOT/in/portfolio_quicktest/txns.csv | python ./table_2_html.py >> $OUT_FILE
	echo "<h3>Holdings</h3>" >> $OUT_FILE
	cat $DATA_ROOT/in/portfolio_quicktest/holdings.csv | python ./table_2_html.py >> $OUT_FILE
	echo '</div>' >> $OUT_FILE

	table2html txns 1_read_txns $SCRIPT_JSON $OUT_FILE portfolio_quicktest
	table2html accounts 1_read_accounts $SCRIPT_JSON $OUT_FILE portfolio_quicktest
	table2html period_holdings 1_read_period_holdings $SCRIPT_JSON $OUT_FILE portfolio_quicktest
	table2html account_txns 2_account_txns_outer $SCRIPT_JSON $OUT_FILE portfolio_quicktest
	table2html account_period_holdings 2_account_period_holdings_outer $SCRIPT_JSON $OUT_FILE portfolio_quicktest
	table2html account_period_activity 3_build_account_period_activity $SCRIPT_JSON $OUT_FILE portfolio_quicktest
	table2html account_period_perf 4_calc_account_period_perf $SCRIPT_JSON $OUT_FILE portfolio_quicktest
	table2html account_period_perf_by_period 5_tag_by_period $SCRIPT_JSON $OUT_FILE portfolio_quicktest
	table2html account_period_perf_by_period_sector 5_tag_by_sector $SCRIPT_JSON $OUT_FILE portfolio_quicktest
	table2html account_period_sector_twr_cagr 6_perf_json_to_columns $SCRIPT_JSON $OUT_FILE portfolio_quicktest

	csv2html "$DATA_ROOT/out/portfolio_quicktest/account_period_sector_perf_baseline.csv" 7_file_account_period_sector_perf $SCRIPT_JSON $OUT_FILE
	csv2html "$DATA_ROOT/out/portfolio_quicktest/account_year_perf_baseline.csv" 7_file_account_year_perf $SCRIPT_JSON $OUT_FILE

	echo '</div></body></head>' >> $OUT_FILE
else
	echo "# portfolio_quicktest script and data" > $OUT_FILE

	echo "## Input files" >> $OUT_FILE
	echo "### Accounts" >> $OUT_FILE
	cat $DATA_ROOT/in/portfolio_quicktest/accounts.csv | python ./table_2_md.py >> $OUT_FILE
	echo "### Transactions" >> $OUT_FILE
	cat $DATA_ROOT/in/portfolio_quicktest/txns.csv | python ./table_2_md.py >> $OUT_FILE
	echo "### Holdings" >> $OUT_FILE
	cat $DATA_ROOT/in/portfolio_quicktest/holdings.csv | python ./table_2_md.py >> $OUT_FILE

	table2md txns 1_read_txns $SCRIPT_JSON $OUT_FILE portfolio_quicktest
	table2md accounts 1_read_accounts $SCRIPT_JSON $OUT_FILE portfolio_quicktest
	table2md period_holdings 1_read_period_holdings $SCRIPT_JSON $OUT_FILE portfolio_quicktest
	table2md account_txns 2_account_txns_outer $SCRIPT_JSON $OUT_FILE portfolio_quicktest
	table2md account_period_holdings 2_account_period_holdings_outer $SCRIPT_JSON $OUT_FILE portfolio_quicktest
	table2md account_period_activity 3_build_account_period_activity $SCRIPT_JSON $OUT_FILE portfolio_quicktest
	table2md account_period_perf 4_calc_account_period_perf $SCRIPT_JSON $OUT_FILE portfolio_quicktest
	table2md account_period_perf_by_period 5_tag_by_period $SCRIPT_JSON $OUT_FILE portfolio_quicktest
	table2md account_period_perf_by_period_sector 5_tag_by_sector $SCRIPT_JSON $OUT_FILE portfolio_quicktest
	table2md account_period_sector_twr_cagr 6_perf_json_to_columns $SCRIPT_JSON $OUT_FILE portfolio_quicktest

	csv2md "$DATA_ROOT/out/portfolio_quicktest/account_period_sector_perf_baseline.csv" 7_file_account_period_sector_perf $SCRIPT_JSON $OUT_FILE
	csv2md "$DATA_ROOT/out/portfolio_quicktest/account_year_perf_baseline.csv" 7_file_account_year_perf $SCRIPT_JSON $OUT_FILE

	echo '</div></body></head>' >> $OUT_FILE
fi