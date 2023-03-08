# This is not a production-quality code for actual financial calculations. Use for testing purposes only.
 
import datetime
import json
from json import JSONEncoder
from dateutil.relativedelta import relativedelta

def wrapped_default(self, obj):
    return getattr(obj.__class__, "__json__", wrapped_default.default)(obj)


wrapped_default.default = JSONEncoder().default

# apply the patch
JSONEncoder.original_default = JSONEncoder.default
JSONEncoder.default = wrapped_default


class CfItQ:
    def __init__(self, d, cf, qty, val):
        self.d = d
        self.cf = cf
        self.qty = qty  # eod qty after cf
        self.val = val  # eod val after cf

    def __json__(self, **options):
        return self.__dict__


class CfIt:
    def __init__(self, d, cf, val):
        self.d = d
        self.cf = cf
        self.val = val  # eod val after cf

    def __json__(self, **options):
        return self.__dict__

class Period:
    def __init__(self, name, start_eod, end_eod):
        self.name = name
        self.start_eod = start_eod
        self.end_eod = end_eod

    def __json__(self, **options):
        return self.__dict__

# Source holdings and txns to ticker-level cf history for all tickers
def txns_and_holdings_to_ticker_cf_history(period_start_eod, period_end_eod, period_start_holdings, all_txns, eod_price_provider):
    ticker_cf_history_map = {}  # ticker->[CfItQ]

    # For each holding, add a period beginning cf record with valid qty and val
    for ticker, qty in period_start_holdings.items():
        # In fact, the cf may have been different on that date, but for our calculations it's irrelevant
        try:
            ticker_cf_history_map[ticker] = [CfItQ(period_start_eod, 0.0, qty, qty * eod_price_provider.get_price(period_start_eod, ticker) if qty != 0 else 0.0)]
        except RuntimeError as re:
            raise RuntimeError(f"cannot append holding for the period start, qty {qty}: {str(re)}")

    # For each txn, add a cf record
    for txn in sorted(all_txns, key=lambda x: x["ts"], reverse=False):
        ticker = txn["t"]
        if ticker not in ticker_cf_history_map:
            # No ticker holdings on the period beginning (see a few lines above), assume zero qty and val
            ticker_cf_history_map[ticker] = [
                CfItQ(period_start_eod, 0.0, 0, 0.0)]

        d = txn["ts"]
        if d <= period_start_eod or d > period_end_eod:
            # Ignore it, it may be used for other period calculations in the same batch
            # Theis is called "all_txns",not "period_txns" for a reason
            continue

        last_cfi = ticker_cf_history_map[ticker][-1]
        if last_cfi.d == d:
            # Recalculate last cf record
            new_qty = last_cfi.qty + txn["q"]
            # cf uses sale price, not eod price
            last_cfi.cf += txn["q"] * txn["p"]
            last_cfi.qty = new_qty
            last_cfi.val = new_qty * eod_price_provider.get_price(d, ticker) if new_qty != 0 else 0.0
        else:
            # Add new cf record
            new_qty = last_cfi.qty + txn["q"]
            # cf uses sale price,not eod price
            ticker_cf_history_map[ticker].append(CfItQ(d, txn["q"] * txn["p"], new_qty, new_qty*eod_price_provider.get_price(d, ticker) if new_qty != 0 else 0.0))

    # For each cf, add a period end cf record with the official EOD price (if needed)
    # We need it to reflect the change in price between last txn and the final day of the period
    for ticker, cf in ticker_cf_history_map.items():
        last_cfi = ticker_cf_history_map[ticker][-1]
        if last_cfi.d == period_end_eod:
            # We have it already covered, end of the period value has been calculated already
            continue

        # Add new cf record
        try:
            ticker_cf_history_map[ticker].append(CfItQ(period_end_eod, 0.0, last_cfi.qty, last_cfi.qty * eod_price_provider.get_price(period_end_eod, ticker) if last_cfi.qty != 0 else 0.0))
        except RuntimeError as re:
            raise RuntimeError(f"cannot append txn for end of period, last cf item date {last_cfi.d} cf {last_cfi.cf} qty {last_cfi.qty} val {last_cfi.val}: {str(re)}")

    return ticker_cf_history_map

# For a set of tickers, group ticker-level cf histories  
def ticker_cf_history_to_group_cf_history(group_ticker_set, ticker_cf_history_map, eod_price_provider):
    group_cf_history = []  # [CfIt]

    # Collect all cashflow dates, we will walk through all of them for all involved tickers
    cashflow_dates = set()
    ticker_date_cf_history_map = {}  # List of cf items to map for faster access
    for ticker, ticker_cf in ticker_cf_history_map.items():
        if ticker not in group_ticker_set:
            continue
        ticker_date_cf_history_map[ticker] = {}

        for ticker_cf_item in ticker_cf:
            cashflow_dates.add(ticker_cf_item.d)
            ticker_date_cf_history_map[ticker][ticker_cf_item.d] = ticker_cf_item

    # Walk through all cashflow dates
    cur_qty_after_cf_map = {k: 0 for k in ticker_date_cf_history_map.keys()}
    for d in sorted(list(cashflow_dates)):
        # Keep track of eod val for each ticker up to this d
        for ticker in ticker_date_cf_history_map.keys():
            if d in ticker_date_cf_history_map[ticker]:
                cur_qty_after_cf_map[ticker] = ticker_date_cf_history_map[ticker][d].qty

        # Accumulate cf and eod val
        total_daily_cf = 0
        total_val_eod_after_cf = 0
        for ticker, ticker_cf_history in ticker_date_cf_history_map.items():
            if d in ticker_cf_history:
                total_daily_cf += ticker_cf_history[d].cf
            total_val_eod_after_cf += cur_qty_after_cf_map[ticker] * eod_price_provider.get_price(d, ticker) if cur_qty_after_cf_map[ticker] != 0 else 0.0
        group_cf_history.append(
            CfIt(d, total_daily_cf, total_val_eod_after_cf))
    return group_cf_history


# For each sector, call  ticker_cf_history_to_group_cf_history
def group_cf_history_by_sector(company_info_provider, ticker_cf_history, eod_price_provider):
    result = {}
    for sector in company_info_provider.get_sectors():
        result[sector] = ticker_cf_history_to_group_cf_history(
            company_info_provider.get_sector_tickers(sector), ticker_cf_history, eod_price_provider)
    return result


def twr_cagr(cf_history):
    prev_cfi = None
    twr = 1.0
    for cfi in cf_history:
        hpr = 0.0
        # Do not calc hpr if prev val after cf is zero, leave zero hpr
        if prev_cfi and abs(prev_cfi.val) > 0.000000001:
            # This may be not the hpr/twr calc methofd you are using, but it is one of those that make sense anyways
            hpr = (cfi.val - cfi.cf) / prev_cfi.val - 1
            twr = twr * (1+hpr)
        prev_cfi = cfi
    d1 = datetime.datetime.strptime(cf_history[0].d, "%Y-%m-%d").date()
    d2 = datetime.datetime.strptime(cf_history[-1].d, "%Y-%m-%d").date()
    years = (d2-d1).days/365
    if twr >= 0:
        cagr = pow(twr, 1/years) - 1.0
    else:
        # Do not allow Python to start complex number calculations
        cagr = -pow(-twr, 1/years) - 1.0
    return twr-1, cagr


# Build ticker-level cf history, build sector cf history, calculate twr/cagr for each sector
def txns_and_holdings_to_twr_cagr_by_sector(period_start_eod, period_end_eod, period_start_holdings, all_txns, eod_price_provider, company_info_provider):
    ticker_cf_history = txns_and_holdings_to_ticker_cf_history(period_start_eod, period_end_eod, period_start_holdings, all_txns, eod_price_provider)
    group_cf_history_map = group_cf_history_by_sector(
        company_info_provider, ticker_cf_history, eod_price_provider)
    sector_perf_map = {}
    for sector, group_cf_history in group_cf_history_map.items():
        if len(group_cf_history) < 2:
            twr = 0.0
            cagr = 0.0
        else:    
            twr, cagr = twr_cagr(group_cf_history)
        sector_perf_map[sector] = {"twr": round(twr, 4), "cagr": round(cagr, 4)}
    return sector_perf_map

# Same as above, but returns json
def txns_and_holdings_to_twr_cagr_by_sector_json(period_start_eod, period_end_eod, period_start_holdings_json, all_txns_json, eod_price_provider, company_info_provider):
    return json.dumps(txns_and_holdings_to_twr_cagr_by_sector(period_start_eod, period_end_eod, json.loads(period_start_holdings_json), json.loads(all_txns_json), eod_price_provider, company_info_provider), sort_keys=True)

# A helper for txns_and_holdings_to_twr_cagr_by_sector_year_quarter
def split_period_into_years_and_quarters(period_start_eod, period_end_eod):
    quarter_periods = []
    year_periods = []
    cur_date = datetime.date.fromisoformat(period_start_eod)
    cur_start_quarter = cur_date
    cur_start_year = cur_date
    while cur_date < datetime.date.fromisoformat(period_end_eod):
        next_eom = cur_date.replace(day=1) + relativedelta(months=2) + datetime.timedelta(days=-1)
        if next_eom.month == 3 or next_eom.month == 6 or next_eom.month == 9 or next_eom.month == 12:
            quarter_periods.append(Period(f'{next_eom.year}Q{next_eom.month // 3}', cur_start_quarter.isoformat(), next_eom.isoformat()))
            cur_start_quarter = next_eom
        if next_eom.month == 12:
            year_periods.append(Period(f'{next_eom.year}', cur_start_year.isoformat(), next_eom.isoformat()))
            cur_start_year = next_eom
        cur_date = next_eom
    return quarter_periods, year_periods

# Helper used below
def accumulate_holdings_by_date(all_holdings, d):
    date_holdings = {}
    for h in all_holdings:
        if h["d"] == d:
            if h["t"] in date_holdings:
                raise RuntimeError(f'holding for ticker {h["t"]} appears more than once for date {d}, first observed qty {date_holdings["t"]}, now got {h["q"]}')
            date_holdings[h["t"]] = h["q"]
    return date_holdings

# Split period into years and quarters and call txns_and_holdings_to_twr_cagr_by_sector
def txns_and_holdings_to_twr_cagr_by_sector_year_quarter(period_start_eod, period_end_eod, all_holdings, all_txns, eod_price_provider, company_info_provider):
    quarter_periods, year_periods = split_period_into_years_and_quarters(period_start_eod, period_end_eod)
    period_perf = {} # All stats returned in a single map, assuming period names (year and quarter) are all distinct
    for period in quarter_periods+year_periods:
        period_start_holdings = accumulate_holdings_by_date(all_holdings, period.start_eod)
        period_perf[period.name] = txns_and_holdings_to_twr_cagr_by_sector(period.start_eod, period.end_eod, period_start_holdings, [txn for txn in all_txns if (period.start_eod < txn["ts"] and txn["ts"] <= period.end_eod)], eod_price_provider, company_info_provider)
    return period_perf

# Same as above, but json
def txns_and_holdings_to_twr_cagr_by_sector_year_quarter_json(period_start_eod, period_end_eod, all_holdings_json, all_txns_json, eod_price_provider, company_info_provider):
    return json.dumps(txns_and_holdings_to_twr_cagr_by_sector_year_quarter(period_start_eod, period_end_eod, json.loads(all_holdings_json), json.loads(all_txns_json), eod_price_provider, company_info_provider), sort_keys=True)
