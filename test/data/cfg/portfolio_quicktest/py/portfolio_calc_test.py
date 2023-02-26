import json
from portfolio_calc import txns_and_holdings_to_ticker_cashflows, ticker_cashflows_to_sector_cashflows, all_sector_cashflows, twr_cagr

test_period_start_eod = "2000-12-31"
test_period_end_eod = "2001-01-31"

class TestSectorInfoProvider:
  def get_sectors():
    return ["All", "Automotive", "Electronics","Software"]

  def get_sector_tickers(sector):
    if sector == "All":
        return ["AAPL","TSLA","MSFT"]
    elif sector == "Electronics":
        return ["AAPL","MSFT"]
    elif sector == "Automotive":
        return ["TSLA"]
    elif sector == "Software":
        return ["MSFT"]
    else:
        return []

class TestEodPriceProvider:
  eod_prices = {
    "2000-12-31":{"TSLA":0.5,"AAPL":1.0, "MSFT":3.0},
    "2001-01-05":{"TSLA":1.0,"AAPL":1.333, "MSFT":3.333},
    "2001-01-10":{"TSLA":1.333,"AAPL":2.0, "MSFT":3.666},
    "2001-01-11":{"TSLA":1.666,"AAPL":2.333, "MSFT":4.0},
    "2001-01-31":{"TSLA":2.0,"AAPL":3.0, "MSFT":5.0}
  }

  def get_price(d,ticker):
    if d not in TestEodPriceProvider.eod_prices or ticker not in TestEodPriceProvider.eod_prices[d]:
      raise RuntimeError(f"{d} price for {ticker} is not available")
    return TestEodPriceProvider.eod_prices[d][ticker]

expected_ticker_cashflows = {
   "AAPL": [
     {"d":"2000-12-31", "val_eod_before_cf":2.0, "cf":0.0, "qty_eod":2},
     {"d":"2001-01-10", "val_eod_before_cf":8.0, "cf":1.8+1.9, "qty_eod":4},
     {"d":"2001-01-31", "val_eod_before_cf":12.0, "cf":0.0, "qty_eod":4}
   ],
   "MSFT": [
    {"d":"2000-12-31", "val_eod_before_cf":3.0, "cf":0.0, "qty_eod":1},
    {"d":"2001-01-11", "val_eod_before_cf":0.0, "cf":-1*3.9, "qty_eod":0},
    {"d":"2001-01-31", "val_eod_before_cf":0.0, "cf":0.0, "qty_eod":0}
   ],
   "TSLA": [
    {"d":"2000-12-31", "val_eod_before_cf":0.0, "cf":0.0, "qty_eod":0},
    {"d":"2001-01-05", "val_eod_before_cf":5.0, "cf":5*0.9, "qty_eod":5},
    {"d":"2001-01-31", "val_eod_before_cf":10.0, "cf":0.0, "qty_eod":5}
   ]}

expected_sector_cashflows = { "All": [
     {"d": "2000-12-31", "val_eod_before_cf":2*TestEodPriceProvider.get_price("2000-12-31","AAPL")+1*TestEodPriceProvider.get_price("2000-12-31","MSFT"), "cf":0.0},
     {"d": "2001-01-05", "val_eod_before_cf":2*TestEodPriceProvider.get_price("2001-01-05","AAPL")+1*TestEodPriceProvider.get_price("2001-01-05","MSFT")+5*TestEodPriceProvider.get_price("2001-01-05","TSLA"), "cf":5*0.9},
     {"d": "2001-01-10", "val_eod_before_cf":4*TestEodPriceProvider.get_price("2001-01-10","AAPL")+1*TestEodPriceProvider.get_price("2001-01-10","MSFT")+5*TestEodPriceProvider.get_price("2001-01-10","TSLA"), "cf":1.8+1.9},
     {"d": "2001-01-11", "val_eod_before_cf":4*TestEodPriceProvider.get_price("2001-01-11","AAPL")+5*TestEodPriceProvider.get_price("2001-01-11","TSLA"), "cf":-1*3.9},
     {"d": "2001-01-31", "val_eod_before_cf":4*TestEodPriceProvider.get_price("2001-01-31","AAPL")+5*TestEodPriceProvider.get_price("2001-01-31","TSLA"), "cf":0.0}
   ],
   "Automotive": [
    {"cf": 0.0, "d": "2000-12-31", "val_eod_before_cf": 0.0},
    {"cf": 4.5, "d": "2001-01-05", "val_eod_before_cf": 5*TestEodPriceProvider.get_price("2001-01-05","TSLA")},
    {"cf": 0.0, "d": "2001-01-31", "val_eod_before_cf": 5*TestEodPriceProvider.get_price("2001-01-31","TSLA")}
  ],
  "Electronics": [
    {"cf": 0.0, "d": "2000-12-31", "val_eod_before_cf": 2*TestEodPriceProvider.get_price("2000-12-31","AAPL")+1*TestEodPriceProvider.get_price("2000-12-31","MSFT")},
    {"cf": 3.7, "d": "2001-01-10", "val_eod_before_cf": 4*TestEodPriceProvider.get_price("2001-01-10","AAPL")+1*TestEodPriceProvider.get_price("2001-01-10","MSFT")},
    {"cf": -3.9, "d": "2001-01-11", "val_eod_before_cf": 4*TestEodPriceProvider.get_price("2001-01-11","AAPL")},
    {"cf": 0.0, "d": "2001-01-31", "val_eod_before_cf": 4*TestEodPriceProvider.get_price("2001-01-31","AAPL")}],
  "Software": [
    {"cf": 0.0, "d": "2000-12-31", "val_eod_before_cf": 1*TestEodPriceProvider.get_price("2000-12-31","MSFT")},
    {"cf": -3.9, "d": "2001-01-11", "val_eod_before_cf": 0.0},
    {"cf": 0.0, "d": "2001-01-31", "val_eod_before_cf": 0.0}
  ]}

def test_txns_and_holdings_to_ticker_cashflows():
  actual = json.dumps(txns_and_holdings_to_ticker_cashflows(
    test_period_start_eod,
    test_period_end_eod,
    {"AAPL": 2, "MSFT": 1},
    [
      {"ts":"2001-01-05 00:01:35.123", "ticker":"TSLA", "qty":5, "price":0.9},
      {"ts":"2001-01-10 00:02:42.963", "ticker":"AAPL", "qty":1, "price":1.8},
      {"ts":"2001-01-10 00:03:42.963", "ticker":"AAPL", "qty":1, "price":1.9},
      {"ts":"2001-01-11 00:03:65.876", "ticker":"MSFT", "qty":-1, "price":3.9}
    ],
    TestEodPriceProvider), sort_keys=True)
  expected = json.dumps(expected_ticker_cashflows, sort_keys=True)

  if actual != expected:
    print(expected)
    print(actual)
  else:
    print ("OK")

def test_ticker_cashflows_to_sector_cashflows():
  actual = json.dumps(all_sector_cashflows(TestSectorInfoProvider, expected_ticker_cashflows, TestEodPriceProvider), sort_keys=True)
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

test_txns_and_holdings_to_ticker_cashflows()
test_ticker_cashflows_to_sector_cashflows()
test_twr_cagr()