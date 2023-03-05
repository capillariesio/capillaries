# 1. Read accounts

We can workonly with accounts that have they first EOM holdings record not later than `period_start_eod`, otherwise we simply cannot calculate their performance. Source data in accounts.csv looks like this:
```
account_id,earliest_period_start
ARKK,2020-12-31
ARKW,2020-12-31
ARKF,2020-12-31
ARKQ,2020-12-31
ARKG,2020-12-31
ARKX,2021-03-31
```
If  `period_start_eod` paramater is set to `2020-12-31`, only first 5 accounts will be loaded.

# 2. Read txns as JSON

Read only transactions that happen after `period_start_eod` and not after `period_end_eod`. We need to convert the data to JSON, so it can be grouped and processed by our Python portfolio performance calculator (remember, Capillaries processors cannot handle more than one row at a time). So, a source transaction from txns.csv
```
ts,account_id,ticker,qty,price
2021-03-30 01:18:07,ARKG,MASS,100453,44.99
```
becomes

| account_id | txn_json |
| ---------- | -------- |
| ARKG | {"ts": "2021-03-30 01:18:07", "ticker": "MASS", "qty": "100453", "price": "44.99"} |

# 3. Group transactions by account

Left outer join accounts with txns into account_txns via string_agg() function with "," separator so the result is only 5 records (one record per account) that look like this:

| account_id | txns_json |
| ---------- | --------- |
| ARKG | {"ts":"2021-01-19 00:20:06","ticker":"BMY","qty":"328990","price":"66.74"},{"ts":"2021-01-08 02:09:04","ticker":"REGN","qty":"27300","price":"498.73"}, ... |

# 4. Read period start holdings as JSON

For the date of `period_start_eod`, read each holding for all accounts:

| account_id | holding_json |
| ---------- | ------------ |
| ARKW       | {"ticker":"SNAP","qty":2378023} |

# 5. Group holdings by account

Left outer join accounts with holdings into account_period_eom_holdings via string_agg() function with "," separator so the result is only 5 records (one record per account) that look like this:

| account_id | holdings_json |
| ---------- | ------------- |
| ARKW | {"ticker":"SNAP","qty":2378023}, {"ticker":"ROKU","qty":987824}, ...

# 6. Combine period start holdings and txns








Join account_period_eom_holdings/account_period_txns->account_period_activity

| account_id | holdings_json | txns_json |
| ---------- | ------------- |----------- |
| ARKF | {"ticker":"WDAY","qty":163133},{"ticker":"JD","qty":270154},{"ticker":"SI","qty":617539}, ...  | {"ts":"2021-03-24 01:52:26","ticker":"PDD","qty":78761,"price":136.06},{"ts":"2021-02-24 02:37:37","ticker":"SQ","qty":97051,"price":237.32}, ...

# acc_number: 12345, holdings: {"ticker":"AAPL", "qty":2},{"ticker":"MSFT", "qty":1}, txn: {"ts":"2001-01-05 00:01:35.123", "ticker":"TSLA", "qty":5, "price":1.00},{"ts":"2001-01-10 00:02:42.123", "ticker":"APPL", "qty":2, "price":2.00}
# acc_number: 98765, holdings: {"ticker":"XOM", "qty":2} txn:

# 7. py_calc: produce twr numbers in accounts_period_perf
# acc_number: 12345, {"All": {"twr": 0.2345, "cagr": 0.1432}, "Automotive": {"twr": 0.2345, "cagr": 0.1432}}
# acc_number: 98765, {"All": {"twr": 0.2345, "cagr": 0.1432}, "Automotive": {"twr": 0.0, "cagr": 0.0}}

# 8. denormalize_and_tag: now we split one record of accounts_period_perf into 4 and tag in sector field to accounts_period_perf_tagged
# acc_number: 12345, sector: All, {"All": {"twr": 0.2345, "cagr": 0.1432}, "Automotive": {"twr": 0.2345, "cagr": 0.1432}}
# acc_number: 12345, sector: Automotive, {"All": {"twr": 0.2345, "cagr": 0.1432}, "Automotive": {"twr": 0.2345, "cagr": 0.1432}}
# acc_number: 98765, sector: All, {"All": {"twr": 0.2345, "cagr": 0.1432}, "Automotive": {"twr": 0.0, "cagr": 0.0}}
# acc_number: 98765, sector: Automotive, {"All": {"twr": 0.2345, "cagr": 0.1432}, "Automotive": {"twr": 0.0, "cagr": 0.0}}

# 9. py_calc accounts_period_perf_tagged -> accounts_period_perf_normalized
# acc_number: 12345, sector: All, twr: 0.2345, cagr: 0.1432
# acc_number: 12345, sector: Automotive, twr: 0.2345, cagr: 0.1432
# acc_number: 98765, sector: All, twr: 0.2345, cagr": 0.1432
# acc_number: 98765, sector: Automotive, twr: 0.0, cagr: 0.0

# 10. Save to file
