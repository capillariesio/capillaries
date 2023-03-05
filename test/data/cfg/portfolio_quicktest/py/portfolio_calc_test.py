import json
from portfolio_calc import txns_and_holdings_to_ticker_cf_history, ticker_cf_history_to_group_cf_history, group_cf_history_by_sector, twr_cagr, txns_and_holdings_to_twr_cagr_by_sector, CfItQ, CfIt
from portfolio_test_eod_price_provider import PortfolioTestEodPriceProvider


test_period_start_eod = "2000-12-31"
test_period_end_eod = "2001-01-31"
test_period_start_holdings = {"AAPL": 2, "MSFT": 1}
test_period_txns = [
    {"ts": "2001-01-05", "ticker": "TSLA", "qty": 5, "price": 0.9},
    {"ts": "2001-01-10", "ticker": "AAPL", "qty": 1, "price": 1.8},
    {"ts": "2001-01-10", "ticker": "AAPL", "qty": 1, "price": 1.9},
    {"ts": "2001-01-11", "ticker": "MSFT", "qty": -1, "price": 3.9}
]


def _test(expected, actual):
    if actual != expected:
        print(expected)
        print(actual)
    else:
        print("OK")


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
        txns_and_holdings_to_twr_cagr_by_sector(
            test_period_start_eod,
            test_period_end_eod,
            json.dumps(test_period_start_holdings),
            json.dumps(test_period_txns),
            UnitTestEodPriceProvider,
            UnitTestCompanyInfoProvider))


def test_price_provider():
    # "AMZN":[{"d":"2020-10-16","p":3272.71},{"d":"2020-10-19","p":3207.21}
    _test(
        round(3272.71 + 2/3*(3207.21-3272.71), 2),
        PortfolioTestEodPriceProvider.get_price("2020-10-18", "AMZN"))


test_txns_and_holdings_to_ticker_cf_history()
test_ticker_cf_history_to_group_cf_history()
test_txns_and_holdings_to_twr_cagr_by_sector()
test_price_provider()
test_twr_cagr()

baba_period_start_eod = "2020-12-31"
baba_period_end_eod = "2021-12-31"
baba_holdings = {} #{"BABA":286964}
baba_txns = [{"ts":"2021-02-09","ticker":"BABA","qty":5400,"price":266.49},
{"ts":"2021-02-10","ticker":"BABA","qty":1980,"price":267.79},
{"ts":"2021-02-11","ticker":"BABA","qty":2700,"price":268.93},
{"ts":"2021-02-12","ticker":"BABA","qty":1800,"price":267.85},
{"ts":"2021-02-16","ticker":"BABA","qty":2880,"price":270.7},
{"ts":"2021-02-17","ticker":"BABA","qty":1800,"price":270.83},
{"ts":"2021-02-19","ticker":"BABA","qty":3600,"price":263.59},
{"ts":"2021-02-22","ticker":"BABA","qty":720,"price":254},
{"ts":"2021-02-23","ticker":"BABA","qty":-1260,"price":252.75},
{"ts":"2021-02-24","ticker":"BABA","qty":-1620,"price":250.34},
{"ts":"2021-02-25","ticker":"BABA","qty":-1440,"price":240.18},
{"ts":"2021-02-26","ticker":"BABA","qty":-1800,"price":237.76},
{"ts":"2021-03-02","ticker":"BABA","qty":720,"price":234.42},
{"ts":"2021-03-03","ticker":"BABA","qty":-1080,"price":236.27},
{"ts":"2021-03-04","ticker":"BABA","qty":-2880,"price":230.5},
{"ts":"2021-03-05","ticker":"BABA","qty":-4320,"price":233.89},
{"ts":"2021-03-08","ticker":"BABA","qty":-2520,"price":226.69},
{"ts":"2021-03-09","ticker":"BABA","qty":1620,"price":238.14},
{"ts":"2021-03-11","ticker":"BABA","qty":360,"price":240.8},
{"ts":"2021-03-12","ticker":"BABA","qty":1440,"price":231.87},
{"ts":"2021-03-15","ticker":"BABA","qty":180,"price":230.28},
{"ts":"2021-03-16","ticker":"BABA","qty":180,"price":226.93},
{"ts":"2021-03-19","ticker":"BABA","qty":-180,"price":239.79},
{"ts":"2021-03-22","ticker":"BABA","qty":-180,"price":237.12},
{"ts":"2021-03-24","ticker":"BABA","qty":-720,"price":229.59},
{"ts":"2021-03-26","ticker":"BABA","qty":-180,"price":227.26},
{"ts":"2021-03-29","ticker":"BABA","qty":-720,"price":231.86},
{"ts":"2021-03-31","ticker":"BABA","qty":720,"price":226.73},
{"ts":"2021-04-05","ticker":"BABA","qty":720,"price":225.3},
{"ts":"2021-04-07","ticker":"BABA","qty":360,"price":225.42},
{"ts":"2021-04-16","ticker":"BABA","qty":-180,"price":238.69},
{"ts":"2021-04-20","ticker":"BABA","qty":-900,"price":229.88},
{"ts":"2021-04-21","ticker":"BABA","qty":-724,"price":229.44},
{"ts":"2021-04-22","ticker":"BABA","qty":180,"price":229.35},
{"ts":"2021-04-23","ticker":"BABA","qty":-540,"price":232.08},
{"ts":"2021-04-26","ticker":"BABA","qty":-180,"price":232.7},
{"ts":"2021-04-28","ticker":"BABA","qty":-180,"price":236.72},
{"ts":"2021-04-29","ticker":"BABA","qty":-540,"price":234.18},
{"ts":"2021-04-30","ticker":"BABA","qty":-180,"price":230.95},
{"ts":"2021-05-03","ticker":"BABA","qty":-1800,"price":230.71},
{"ts":"2021-05-04","ticker":"BABA","qty":-900,"price":227.9},
{"ts":"2021-05-05","ticker":"BABA","qty":-720,"price":226.78},
{"ts":"2021-05-06","ticker":"BABA","qty":-1260,"price":226.42},
{"ts":"2021-05-07","ticker":"BABA","qty":-900,"price":225.31},
{"ts":"2021-05-10","ticker":"BABA","qty":-1260,"price":219.53},
{"ts":"2021-05-11","ticker":"BABA","qty":-720,"price":221.38},
{"ts":"2021-05-12","ticker":"BABA","qty":-1260,"price":219.9},
{"ts":"2021-05-13","ticker":"BABA","qty":-720,"price":206.08},
{"ts":"2021-05-14","ticker":"BABA","qty":-360,"price":209.51},
{"ts":"2021-05-17","ticker":"BABA","qty":-900,"price":211.05},
{"ts":"2021-05-18","ticker":"BABA","qty":-720,"price":213.72},
{"ts":"2021-05-19","ticker":"BABA","qty":-540,"price":212.54},
{"ts":"2021-05-24","ticker":"BABA","qty":360,"price":210.44},
{"ts":"2021-06-03","ticker":"BABA","qty":-180,"price":217.04},
{"ts":"2021-06-07","ticker":"BABA","qty":-180,"price":216.9},
{"ts":"2021-06-09","ticker":"BABA","qty":-180,"price":213.32},
{"ts":"2021-06-10","ticker":"BABA","qty":-180,"price":213.07},
{"ts":"2021-06-11","ticker":"BABA","qty":-180,"price":211.64},
{"ts":"2021-06-21","ticker":"BABA","qty":27574,"price":211.06},
{"ts":"2021-06-23","ticker":"BABA","qty":219,"price":214.86},
{"ts":"2021-06-25","ticker":"BABA","qty":438,"price":228.5},
{"ts":"2021-06-28","ticker":"BABA","qty":-219,"price":228.59},
{"ts":"2021-06-30","ticker":"BABA","qty":438,"price":226.78},
{"ts":"2021-07-01","ticker":"BABA","qty":-219,"price":221.87},
{"ts":"2021-07-06","ticker":"BABA","qty":-219,"price":211.6},
{"ts":"2021-07-07","ticker":"BABA","qty":-438,"price":208},
{"ts":"2021-07-09","ticker":"BABA","qty":-219,"price":205.94},
{"ts":"2021-07-14","ticker":"BABA","qty":-438,"price":211.5},
{"ts":"2021-07-15","ticker":"BABA","qty":-657,"price":214.76},
{"ts":"2021-07-16","ticker":"BABA","qty":-438,"price":212.1},
{"ts":"2021-07-19","ticker":"BABA","qty":-657,"price":208.91},
{"ts":"2021-07-20","ticker":"BABA","qty":-219,"price":210.59},
{"ts":"2021-07-21","ticker":"BABA","qty":-219,"price":211.08},
{"ts":"2021-07-23","ticker":"BABA","qty":-657,"price":206.53},
{"ts":"2021-07-27","ticker":"BABA","qty":-657,"price":186.07},
{"ts":"2021-07-28","ticker":"BABA","qty":-219,"price":196.01},
{"ts":"2021-07-29","ticker":"BABA","qty":-219,"price":197.54},
{"ts":"2021-07-30","ticker":"BABA","qty":-219,"price":195.19},
{"ts":"2021-08-02","ticker":"BABA","qty":-1314,"price":200.09},
{"ts":"2021-08-03","ticker":"BABA","qty":-1095,"price":197.38},
{"ts":"2021-08-04","ticker":"BABA","qty":-219,"price":200.71},
{"ts":"2021-08-05","ticker":"BABA","qty":-219,"price":199.28},
{"ts":"2021-08-11","ticker":"BABA","qty":-440,"price":194.86},
{"ts":"2021-08-12","ticker":"BABA","qty":-219,"price":191.66},
{"ts":"2021-08-13","ticker":"BABA","qty":-219,"price":188.62},
{"ts":"2021-08-16","ticker":"BABA","qty":-438,"price":182.71},
{"ts":"2021-08-17","ticker":"BABA","qty":-438,"price":173.73},
{"ts":"2021-08-18","ticker":"BABA","qty":-220,"price":172.35},
{"ts":"2021-08-19","ticker":"BABA","qty":-880,"price":160.55},
{"ts":"2021-08-20","ticker":"BABA","qty":-219,"price":157.96},
{"ts":"2021-08-23","ticker":"BABA","qty":-219,"price":161.06},
{"ts":"2021-08-24","ticker":"BABA","qty":-67799,"price":171.7},
{"ts":"2021-08-25","ticker":"BABA","qty":-117,"price":169.1},
{"ts":"2021-08-26","ticker":"BABA","qty":-47017,"price":165.24},
{"ts":"2021-08-27","ticker":"BABA","qty":-13639,"price":159.47},
{"ts":"2021-08-30","ticker":"BABA","qty":-15141,"price":162.29},
{"ts":"2021-08-31","ticker":"BABA","qty":-3,"price":166.99},
{"ts":"2021-09-01","ticker":"BABA","qty":-1376,"price":173.28},
{"ts":"2021-09-01","ticker":"BABA","qty":-300,"price":173.28}]

print(txns_and_holdings_to_ticker_cf_history(
            baba_period_start_eod,
            baba_period_end_eod,
            baba_holdings,
            baba_txns,
            PortfolioTestEodPriceProvider))