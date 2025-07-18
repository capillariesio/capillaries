import json
from portfolio_calc import txns_and_holdings_to_ticker_cf_history, ticker_cf_history_to_group_cf_history, group_cf_history_by_sector, twr_cagr, txns_and_holdings_to_twr_cagr_by_sector_json, CfItQ, CfIt, split_period_into_years_and_quarters, txns_and_holdings_to_twr_cagr_by_sector_year_quarter_json
from portfolio_test_eod_price_provider import PortfolioTestEodPriceProvider
from portfolio_test_company_info_provider import PortfolioTestCompanyInfoProvider


test_period_start_eod = "2000-12-31"
test_period_end_eod = "2001-01-31"
test_period_start_holdings = {"AAPL": 2, "MSFT": 1}
test_period_txns = [
    {"ts": "2001-01-05", "t": "TSLA", "q":  5, "p": 0.9},
    {"ts": "2001-01-10", "t": "AAPL", "q":  1, "p": 1.8},
    {"ts": "2001-01-10", "t": "AAPL", "q":  1, "p": 1.9},
    {"ts": "2001-01-11", "t": "MSFT", "q": -1, "p": 3.9}
]

def _test(expected, actual):
    if actual != expected:
        print(expected)
        print(actual)
    else:
        print("OK")

# Simulates 1_read_txns packing
# fmt.Sprintf(`\"%s|%s|%d|%s\"`, r.col_ts, r.col_ticker, r.col_qty, decimal2(r.col_price))
def pack_txn(txn):
    return f'{txn["ts"]}|{txn["t"]}|{txn["q"]}|{txn["p"]}'

# Simulates 1_read_txns packing
# fmt.Sprintf(`\"%s|%s|%d|%s\"`, r.col_ts, r.col_ticker, r.col_qty, decimal2(r.col_price))
def pack_holding(h):
    return f'{h["d"]}|{h["t"]}|{h["q"]}'

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
        "TSLA": {"sectors": ["Automotive"]},
        "MSFT": {"sectors": ["Electronics", "Software"]}
    }


class UnitTestEodPriceProvider:
    eod_prices = {
        "2000-12-31": {"AAPL": 1.0,   "TSLA": 0.5,   "MSFT": 3.0},
        "2001-01-05": {"AAPL": 1.333, "TSLA": 1.0,   "MSFT": 3.333},
        "2001-01-10": {"AAPL": 2.0,   "TSLA": 1.333, "MSFT": 3.666},
        "2001-01-11": {"AAPL": 2.333, "TSLA": 1.666, "MSFT": 4.0},
        "2001-01-12": {"AAPL": 4.0,   "TSLA": 3.0,   "MSFT": 6.0},
        "2001-01-31": {"AAPL": 3.0,   "TSLA": 2.0,   "MSFT": 5.0}
    }

    def get_price(d, ticker):
        if d not in UnitTestEodPriceProvider.eod_prices or ticker not in UnitTestEodPriceProvider.eod_prices[d]:
            raise RuntimeError(f"{d} price for {ticker} is not available")
        return UnitTestEodPriceProvider.eod_prices[d][ticker]


expected_ticker_cashflows = {
    "AAPL": [
        CfItQ("2000-12-31", 0.0, 2,  2.0),
        CfItQ("2001-01-10", 3.7, 4,  8.0),
        CfItQ("2001-01-31", 0.0, 4, 12.0),
    ],
    "MSFT": [
        CfItQ("2000-12-31",  0.0, 1, 3.0),
        CfItQ("2001-01-11", -3.9, 0, 0.0),
        CfItQ("2001-01-31",  0.0, 0, 0.0)
    ],
    "TSLA": [
        CfItQ("2000-12-31", 0.0, 0,  0.0),
        CfItQ("2001-01-05", 4.5, 5,  5.0),
        CfItQ("2001-01-31", 0.0, 5, 10.0)
    ]
}

expected_sector_cashflows = {
    "All": [
        CfIt("2000-12-31",  0.0,   2*1.0 + 0 + 1*3.0),
        CfIt("2001-01-05",  4.5, 2*1.333 + 5*1.0 + 1*3.333),
        CfIt("2001-01-10",  3.7,   4*2.0 + 5*1.333 + 1*3.666),
        CfIt("2001-01-11", -3.9, 4*2.333 + 5*1.666 + 0),
        CfIt("2001-01-31",  0.0,   4*3.0 + 5*2.0 + 0)
    ],
    "Automotive": [
        CfIt("2000-12-31", 0.0,   0.0),
        CfIt("2001-01-05", 4.5, 5*1.0),
        CfIt("2001-01-31", 0.0, 5*2.0)
    ],
    "Electronics": [
        CfIt("2000-12-31",  0.0,   2*1.0 + 0 + 1*3.0),
        CfIt("2001-01-10",  3.7,   4*2.0 + 0 + 1*3.666),
        CfIt("2001-01-11", -3.9, 4*2.333 + 0 + 0),
        CfIt("2001-01-31",  0.0,   4*3.0 + 0 + 0)
    ],
    "Software": [
        CfIt("2000-12-31",  0.0, 0 + 0 + 1*3.0),
        CfIt("2001-01-11", -3.9, 0 + 0 + 0.0),
        CfIt("2001-01-31",  0.0, 0 + 0 + 0.0)
    ]}


def test_txns_and_holdings_to_ticker_cf_history():
    _test(
        json.dumps(expected_ticker_cashflows, sort_keys=True),
        json.dumps(txns_and_holdings_to_ticker_cf_history(
            test_period_start_eod,
            test_period_end_eod,
            test_period_start_holdings,
            test_period_txns,
            UnitTestEodPriceProvider), sort_keys=True))


def test_ticker_cf_history_to_group_cf_history():
    _test(
        json.dumps(expected_sector_cashflows, sort_keys=True),
        json.dumps(group_cf_history_by_sector(
            UnitTestCompanyInfoProvider,
            expected_ticker_cashflows,
            UnitTestEodPriceProvider), sort_keys=True))


def test_twr_cagr():
    # https://www.fool.com/about/how-to-calculate-investment-returns/
    actual_twr, actual_cagr = twr_cagr([
        CfIt("2015-12-31", 0.0, 20300),
        CfIt("2016-02-27", 500.0, 22273),
        CfIt("2016-05-16", 500.0, 24437),
        CfIt("2016-08-12", -250.0, 22573),
        CfIt("2016-10-15", 500.0, 25018),
        CfIt("2017-12-31", 0.0, 25992)])
    actual_twr = round(actual_twr, 4)
    actual_cagr = round(actual_cagr, 4)
    _test((0.2148, 0.1021), (actual_twr, actual_cagr))


expected_sector_perf_map = json.dumps({
    "All":         {"cagr":  56628.415, "twr": 1.5333},
    "Automotive":  {"cagr":  3501.5588, "twr":    1.0},
    "Electronics": {"cagr": 20485.9885, "twr": 1.3237},
    "Software":    {"cagr":    20.9579, "twr":    0.3}},
    sort_keys=True)


def test_txns_and_holdings_to_twr_cagr_by_sector():
    _test(
        expected_sector_perf_map,
        txns_and_holdings_to_twr_cagr_by_sector_json(
            test_period_start_eod,
            test_period_end_eod,
            json.dumps(test_period_start_holdings),
            json.dumps([pack_txn(x) for x in test_period_txns]),
            UnitTestEodPriceProvider,
            UnitTestCompanyInfoProvider))


def test_price_provider():
    # "AMZN":[{"d":"2020-10-16","p":3272.71},{"d":"2020-10-19","p":3207.21}
    _test(
        round(3272.71 + 2/3*(3207.21-3272.71), 2),
        PortfolioTestEodPriceProvider.get_price("2020-10-18", "AMZN"))
    
def test_split_period_into_years_and_quarters():
    quarter_periods, year_periods = split_period_into_years_and_quarters("2020-12-31","2022-12-31")
    _test(json.dumps(quarter_periods),json.dumps([{"name": "2021Q1", "start_eod": "2020-12-31", "end_eod": "2021-03-31"}, {"name": "2021Q2", "start_eod": "2021-03-31", "end_eod": "2021-06-30"}, {"name": "2021Q3", "start_eod": "2021-06-30", "end_eod": "2021-09-30"}, {"name": "2021Q4", "start_eod": "2021-09-30", "end_eod": "2021-12-31"}, {"name": "2022Q1", "start_eod": "2021-12-31", "end_eod": "2022-03-31"}, {"name": "2022Q2", "start_eod": "2022-03-31", "end_eod": "2022-06-30"}, {"name": "2022Q3", "start_eod": "2022-06-30", "end_eod": "2022-09-30"}, {"name": "2022Q4", "start_eod": "2022-09-30", "end_eod": "2022-12-31"}]))
    _test(json.dumps(year_periods),json.dumps([{"name": "2021", "start_eod": "2020-12-31", "end_eod": "2021-12-31"}, {"name": "2022", "start_eod": "2021-12-31", "end_eod": "2022-12-31"}]))

def test_txns_and_holdings_to_twr_cagr_by_sector_year_quarter():
    holdings_to_test = [{"d":"2000-12-31", "t":"AAPL", "q":2}, {"d":"2000-12-31", "t":"MSFT", "q":1}]
    txns_to_test = test_period_txns + [{"ts": "2001-01-12", "t": "AAPL", "q": -4, "p": 4.0}, {"ts": "2001-01-12", "t": "TSLA", "q": -5, "p": 2.0}]
    _test(
        json.dumps({"2001": {"All": {"cagr": 1.9939, "twr": 1.9939}, "Automotive": {"cagr": 1.0, "twr": 1.0}, "Electronics": {"cagr": 2.0983, "twr": 2.0983}, "Software": {"cagr": 0.3, "twr": 0.3}}, "2001Q1": {"All": {"cagr": 84.3871, "twr": 1.9939}, "Automotive": {"cagr": 15.6281, "twr": 1.0}, "Electronics": {"cagr": 97.1207, "twr": 2.0983}, "Software": {"cagr": 1.898, "twr": 0.3}}, "2001Q2": {"All": {"cagr": 0.0, "twr": 0.0}, "Automotive": {"cagr": 0.0, "twr": 0.0}, "Electronics": {"cagr": 0.0, "twr": 0.0}, "Software": {"cagr": 0.0, "twr": 0.0}}, "2001Q3": {"All": {"cagr": 0.0, "twr": 0.0}, "Automotive": {"cagr": 0.0, "twr": 0.0}, "Electronics": {"cagr": 0.0, "twr": 0.0}, "Software": {"cagr": 0.0, "twr": 0.0}}, "2001Q4": {"All": {"cagr": 0.0, "twr": 0.0}, "Automotive": {"cagr": 0.0, "twr": 0.0}, "Electronics": {"cagr": 0.0, "twr": 0.0}, "Software": {"cagr": 0.0, "twr": 0.0}}}),
        txns_and_holdings_to_twr_cagr_by_sector_year_quarter_json(
            "2000-12-31",
            "2001-12-31",
            json.dumps([pack_holding(x) for x in holdings_to_test]),
            json.dumps([pack_txn(x) for x in txns_to_test]),
            UnitTestEodPriceProvider,
            UnitTestCompanyInfoProvider))

# For some more serious troubleshooting
# print(txns_and_holdings_to_twr_cagr_by_sector_year_quarter_json(
#     "2020-12-31",
#     "2022-12-31",
#     '[...long holdings json from cassandra]',
#     '[...long txns from casandra]]',
#     PortfolioTestEodPriceProvider,
#     PortfolioTestCompanyInfoProvider))

test_txns_and_holdings_to_ticker_cf_history()
test_ticker_cf_history_to_group_cf_history()
test_txns_and_holdings_to_twr_cagr_by_sector()
test_price_provider()
test_twr_cagr()
test_split_period_into_years_and_quarters()
test_txns_and_holdings_to_twr_cagr_by_sector_year_quarter()

