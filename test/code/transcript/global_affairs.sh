#!/bin/bash

source ./util.sh

OUT_FILE=transcript_global_affairs.html
DATA_ROOT="../../data"
SCRIPT_JSON="$DATA_ROOT/cfg/global_affairs_quicktest/script.json"

if [[ $OUT_FILE == *html ]]; then
	echo '<html><head><style>' > $OUT_FILE
	cat transcript.css  >> $OUT_FILE
	echo '</style></head><body>' >> $OUT_FILE
	echo '<div class="container">' >> $OUT_FILE

	echo "<div class="row"><h1>global_affairs_quicktest script and data</h1></div>" >> $OUT_FILE

	echo '<div class="row">' >> $OUT_FILE
	echo "<h2>Input files</h2>" >> $OUT_FILE
	echo "<h3>Global Affairs projects financials</h3>" >> $OUT_FILE
	cat $DATA_ROOT/in/global_affairs_quicktest/harvested_project_financials_quicktest.csv | python ./table_2_html.py >> $OUT_FILE
	echo '</div>' >> $OUT_FILE

	table2html project_financials 1_read_project_financials $SCRIPT_JSON $OUT_FILE global_affairs_quicktest
	table2html quarterly_project_bdgt 2_calc_quarterly_budget $SCRIPT_JSON $OUT_FILE global_affairs_quicktest
	table2html quarterly_project_bdgt_tggd_by_country 3_tag_countries $SCRIPT_JSON $OUT_FILE global_affairs_quicktest
	table2html quarterly_project_bdgt_tggd_by_sector 3_tag_sectors $SCRIPT_JSON $OUT_FILE global_affairs_quicktest
	table2html quarterly_project_bdgt_tggd_by_country_qtr 4_tag_countries_quarter $SCRIPT_JSON $OUT_FILE global_affairs_quicktest
	table2html quarterly_project_bdgt_tggd_by_sector_qtr 4_tag_sectors_quarter $SCRIPT_JSON $OUT_FILE global_affairs_quicktest
	table2html quarterly_project_bdgt_tggd_by_partner_qtr 4_tag_partners_quarter $SCRIPT_JSON $OUT_FILE global_affairs_quicktest
	table2html project_country_quarter_amt 5_project_country_quarter_amt $SCRIPT_JSON $OUT_FILE global_affairs_quicktest
	table2html project_sector_quarter_amt 5_project_sector_quarter_amt $SCRIPT_JSON $OUT_FILE global_affairs_quicktest
	table2html project_country_quarter_amt 5_project_partner_quarter_amt $SCRIPT_JSON $OUT_FILE global_affairs_quicktest

	csv2html "$DATA_ROOT/out/global_affairs_quicktest/project_country_quarter_amt_baseline.csv" 6_file_project_country_quarter_amt $SCRIPT_JSON $OUT_FILE
	csv2html "$DATA_ROOT/out/global_affairs_quicktest/project_sector_quarter_amt_baseline.csv" 6_file_project_sector_quarter_amt $SCRIPT_JSON $OUT_FILE
	csv2html "$DATA_ROOT/out/global_affairs_quicktest/project_partner_quarter_amt_baseline.csv" 6_file_project_partner_quarter_amt $SCRIPT_JSON $OUT_FILE

	echo '</div></body></head>' >> $OUT_FILE
else
	echo "# global_affairs_quicktest script and data" > $OUT_FILE

	echo "## Input files" >> $OUT_FILE
	echo "### Global Affairs projects financials" >> $OUT_FILE
	cat $DATA_ROOT/in/global_affairs_quicktest/harvested_project_financials_quicktest.csv | python ./table_2_md.py >> $OUT_FILE

	table2md project_financials 1_read_project_financials $SCRIPT_JSON $OUT_FILE global_affairs_quicktest
	table2md quarterly_project_bdgt 2_calc_quarterly_budget $SCRIPT_JSON $OUT_FILE global_affairs_quicktest
	table2md quarterly_project_bdgt_tggd_by_country 3_tag_countries $SCRIPT_JSON $OUT_FILE global_affairs_quicktest
	table2md quarterly_project_bdgt_tggd_by_sector 3_tag_sectors $SCRIPT_JSON $OUT_FILE global_affairs_quicktest
	table2md quarterly_project_bdgt_tggd_by_country_qtr 4_tag_countries_quarter $SCRIPT_JSON $OUT_FILE global_affairs_quicktest
	table2md quarterly_project_bdgt_tggd_by_sector_qtr 4_tag_sectors_quarter $SCRIPT_JSON $OUT_FILE global_affairs_quicktest
	table2md quarterly_project_bdgt_tggd_by_partner_qtr 4_tag_partners_quarter $SCRIPT_JSON $OUT_FILE global_affairs_quicktest
	table2md project_country_quarter_amt 5_project_country_quarter_amt $SCRIPT_JSON $OUT_FILE global_affairs_quicktest
	table2md project_sector_quarter_amt 5_project_sector_quarter_amt $SCRIPT_JSON $OUT_FILE global_affairs_quicktest
	table2md project_country_quarter_amt 5_project_partner_quarter_amt $SCRIPT_JSON $OUT_FILE global_affairs_quicktest

	csv2md "$DATA_ROOT/out/global_affairs_quicktest/project_country_quarter_amt_baseline.csv" 6_file_project_country_quarter_amt $SCRIPT_JSON $OUT_FILE
	csv2md "$DATA_ROOT/out/global_affairs_quicktest/project_sector_quarter_amt_baseline.csv" 6_file_project_sector_quarter_amt $SCRIPT_JSON $OUT_FILE
	csv2md "$DATA_ROOT/out/global_affairs_quicktest/project_partner_quarter_amt_baseline.csv" 6_file_project_partner_quarter_amt $SCRIPT_JSON $OUT_FILE
fi