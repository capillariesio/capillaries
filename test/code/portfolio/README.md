# 1. Read into accounts (only those that were open before or on period_start_date)
# acc_number: 12345, name: John Doe, open_date:
# acc_number: 98765, name: Jane Doe, open_date

# 2. Read into period_txns as JSON (only those between start and end)
# acc_number: 12345, txn: {"ts":"2001-01-05 00:01:35.123", "ticker":"TSLA", "qty":5, "price":1.00}
# acc_number: 12345, txn: {"ts":"2001-01-10 00:02:42.123", "ticker":"APPL", "qty":2, "price":2.00}

# 3. Outer join accounts/period_txns->account_period_txns via string_agg
# acc_number: 12345, txn: {"ts":"2001-01-05 00:01:35.123", "ticker":"TSLA", "qty":5, "price":1.00},{"ts":"2001-01-10 00:02:42.123", "ticker":"APPL", "qty":2, "price":2.00}
# acc_number: 98765, txn:

# 4. Read into period_start_holdings as JSON (for the specific date)
# acc_number: 12345, eod: 2000-12-31, holdings: {"ticker":"AAPL", "qty":2}
# acc_number: 12345, eod: 2000-12-31, holdings: {"ticker":"MSFT", "qty":1}
# acc_number: 98765, eod: 2000-12-31, holdings: {"ticker":"XOM", "qty":10}

# 5. Outer join accounts/period_start_holdings->acount_period_start_holdings via string_agg
# acc_number: 12345, eod: 2000-12-31, holdings: {"ticker":"AAPL", "qty":2},{"ticker":"MSFT", "qty":1}
# acc_number: 98765, eod: 2000-12-31, holdings: {"ticker":"XOM", "qty":10}

# 6. Outer join account_period_start_holdings/account_period_txns->account_period_activity
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
