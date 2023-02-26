import json
from portfolio_calc import txns_and_holdings_to_ticker_cf_history, ticker_cf_history_to_group_cf_history, group_cf_history_by_sector, twr_cagr, txns_and_holdings_to_twr_cagr_by_sector

test_period_start_eod = "2000-12-31"
test_period_end_eod = "2001-01-31"
test_period_start_holdings = {"AAPL": 2, "MSFT": 1}
test_period_txns = [
      {"ts":"2001-01-05 00:01:35.123", "ticker":"TSLA", "qty":5, "price":0.9},
      {"ts":"2001-01-10 00:02:42.963", "ticker":"AAPL", "qty":1, "price":1.8},
      {"ts":"2001-01-10 00:03:42.963", "ticker":"AAPL", "qty":1, "price":1.9},
      {"ts":"2001-01-11 00:03:65.876", "ticker":"MSFT", "qty":-1, "price":3.9}
    ]

class UnitTestCompanyInfoProvider:
  def get_sectors():
    result_sectors = set(["All"])
    for _, ticker_info in UnitTestCompanyInfoProvider.company_info.items():
      for sector in ticker_info["sectors"]:
        result_sectors.add(sector)
    return sorted(list(result_sectors))

  def get_sector_tickers(sector):
    result_tickers = []
    for ticker, ticker_info in UnitTestCompanyInfoProvider.company_info.items():
      if sector in ticker_info["sectors"] or sector == "All":
        result_tickers.append(ticker)
    return sorted(result_tickers)

  company_info = {
      "AAPL": {"sectors": ["Electronics"]},
      "MSFT": {"sectors": ["Electronics","Software"]},
      "TSLA": {"sectors": ["Automotive"]}
  }

class UnitTestEodPriceProvider:
  eod_prices = {
    "2000-12-31":{"TSLA":0.5,"AAPL":1.0, "MSFT":3.0},
    "2001-01-05":{"TSLA":1.0,"AAPL":1.333, "MSFT":3.333},
    "2001-01-10":{"TSLA":1.333,"AAPL":2.0, "MSFT":3.666},
    "2001-01-11":{"TSLA":1.666,"AAPL":2.333, "MSFT":4.0},
    "2001-01-31":{"TSLA":2.0,"AAPL":3.0, "MSFT":5.0}
  }

  def get_price(d,ticker):
    if d not in UnitTestEodPriceProvider.eod_prices or ticker not in UnitTestEodPriceProvider.eod_prices[d]:
      raise RuntimeError(f"{d} price for {ticker} is not available")
    return UnitTestEodPriceProvider.eod_prices[d][ticker]

expected_ticker_cashflows = {
    "AAPL": [
        {
            "cf": 0.0,
            "d": "2000-12-31",
            "qty_eod_after_cf": 2,
            "qty_eod_before_cf": 2,
            "val_eod_before_cf": 2.0
        },
        {
            "cf": 3.7,
            "d": "2001-01-10",
            "qty_eod_after_cf": 4,
            "qty_eod_before_cf": 2,
            "val_eod_before_cf": 4.0
        },
        {
            "cf": 0.0,
            "d": "2001-01-31",
            "qty_eod_after_cf": 4,
            "qty_eod_before_cf": 4,
            "val_eod_before_cf": 12.0
        }
    ],
    "MSFT": [
        {
            "cf": 0.0,
            "d": "2000-12-31",
            "qty_eod_after_cf": 1,
            "qty_eod_before_cf": 1,
            "val_eod_before_cf": 3.0
        },
        {
            "cf": -3.9,
            "d": "2001-01-11",
            "qty_eod_after_cf": 0,
            "qty_eod_before_cf": 1,
            "val_eod_before_cf": 4.0
        },
        {
            "cf": 0.0,
            "d": "2001-01-31",
            "qty_eod_after_cf": 0,
            "qty_eod_before_cf": 0,
            "val_eod_before_cf": 0.0
        }
    ],
    "TSLA": [
        {
            "cf": 0.0,
            "d": "2000-12-31",
            "qty_eod_after_cf": 0,
            "qty_eod_before_cf": 0,
            "val_eod_before_cf": 0.0
        },
        {
            "cf": 4.5,
            "d": "2001-01-05",
            "qty_eod_after_cf": 5,
            "qty_eod_before_cf": 0,
            "val_eod_before_cf": 0.0
        },
        {
            "cf": 0.0,
            "d": "2001-01-31",
            "qty_eod_after_cf": 5,
            "qty_eod_before_cf": 5,
            "val_eod_before_cf": 10.0
        }
    ]
}

expected_sector_cashflows = {
  "All": [
    {"cf": 0.0, "d": "2000-12-31", "val_eod_before_cf": 5.0},
    {"cf": 4.5, "d": "2001-01-05", "val_eod_before_cf": 5.9990000000000006},
    {"cf": 3.7, "d": "2001-01-10", "val_eod_before_cf": 7.666},
    {"cf": -3.9, "d": "2001-01-11", "val_eod_before_cf": 8.666},
    {"cf": 0.0, "d": "2001-01-31", "val_eod_before_cf": 22.0}
  ],
  "Automotive": [
    {"cf": 0.0, "d": "2000-12-31", "val_eod_before_cf": 0.0},
    {"cf": 4.5, "d": "2001-01-05", "val_eod_before_cf": 0.0},
    {"cf": 0.0, "d": "2001-01-31", "val_eod_before_cf": 10.0}
  ],
  "Electronics": [
    {"cf": 0.0, "d": "2000-12-31", "val_eod_before_cf": 5.0},
    {"cf": 3.7, "d": "2001-01-10", "val_eod_before_cf": 7.666},
    {"cf": -3.9, "d": "2001-01-11", "val_eod_before_cf": 8.666},
    {"cf": 0.0, "d": "2001-01-31", "val_eod_before_cf": 12.0}
  ],
  "Software": [
    {"cf": 0.0, "d": "2000-12-31", "val_eod_before_cf": 3.0},
    {"cf": -3.9, "d": "2001-01-11", "val_eod_before_cf": 4.0},
    {"cf": 0.0, "d": "2001-01-31", "val_eod_before_cf": 0.0}
  ]}

def test_txns_and_holdings_to_ticker_cf_history():
  actual = json.dumps(txns_and_holdings_to_ticker_cf_history(
    test_period_start_eod,
    test_period_end_eod,
    test_period_start_holdings,
    test_period_txns,
    UnitTestEodPriceProvider), sort_keys=True)
  expected = json.dumps(expected_ticker_cashflows, sort_keys=True)

  if actual != expected:
    print(expected)
    print(actual)
  else:
    print ("OK")

def test_ticker_cf_history_to_group_cf_history():
  actual = json.dumps(group_cf_history_by_sector(UnitTestCompanyInfoProvider, expected_ticker_cashflows, UnitTestEodPriceProvider), sort_keys=True)
  expected = json.dumps(expected_sector_cashflows, sort_keys=True)

  if actual != expected:
    print(expected)
    print(actual)
  else:
    print ("OK")

def test_twr_cagr():
  # https://www.fool.com/about/how-to-calculate-investment-returns/
  actual_twr, actual_cagr = twr_cagr([
    {"d": "2015-12-31", "val_eod_before_cf":20300, "cf":0.0},
    {"d": "2016-02-27", "val_eod_before_cf":21773, "cf":500.0},
    {"d": "2016-05-16", "val_eod_before_cf":23937, "cf":500.0},
    {"d": "2016-08-12", "val_eod_before_cf":22823, "cf":-250.0},
    {"d": "2016-10-15", "val_eod_before_cf":24518, "cf":500.0},
    {"d": "2017-12-31", "val_eod_before_cf":25992, "cf":0.0}
  ])
  actual_twr = round(actual_twr,4)
  actual_cagr= round(actual_cagr,4)
  expected_twr = 0.2148
  expected_cagr = 0.1021

  if actual_twr != expected_twr or actual_cagr != expected_cagr:
    print(expected_twr, expected_cagr)
    print(actual_twr, actual_cagr)
  else:
    print ("OK")

expected_sector_perf_map = json.dumps({
  "All": {"cagr": 572388.2529, "twr": 2.0833},
  "Automotive": {"cagr": 12108.9676, "twr": 1.2222},
  "Electronics": {"cagr": 331266.0104, "twr": 1.9433},
  "Software": {"cagr": 28.5837, "twr": 0.3333}},
  sort_keys=True)

def test_txns_and_holdings_to_twr_cagr_by_sector():
  sector_perf_map = txns_and_holdings_to_twr_cagr_by_sector(
    test_period_start_eod,
    test_period_end_eod,
    json.dumps(test_period_start_holdings),
    json.dumps(test_period_txns),
    UnitTestEodPriceProvider,
    UnitTestCompanyInfoProvider)
  if sector_perf_map != expected_sector_perf_map:
    print(expected_sector_perf_map)
    print(sector_perf_map)
  else:
    print ("OK")
  

test_txns_and_holdings_to_ticker_cf_history()
test_ticker_cf_history_to_group_cf_history()
test_twr_cagr()
test_txns_and_holdings_to_twr_cagr_by_sector()