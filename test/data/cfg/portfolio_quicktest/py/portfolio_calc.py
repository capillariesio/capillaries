import datetime, json

def txns_and_holdings_to_ticker_cashflows(period_start_eod, period_end_eod, period_start_holdings, period_txns, eod_price_provider):
  ticker_cf_history_map = {}

  # For each holding, add a period beginning cf record

  for ticker, qty in period_start_holdings.items():
    ticker_cf_history_map[ticker] = [{
      "d": period_start_eod,
      "val_eod_before_cf":qty * eod_price_provider.get_price(period_start_eod,ticker),
      "cf": 0.0,
      "qty_eod_before_cf":qty,
      "qty_eod_after_cf":qty}]

  # For each txn, add a cf record
  for txn in sorted(period_txns, key=lambda x: x["ts"], reverse=False):
    ticker = txn["ticker"]
    if ticker not in ticker_cf_history_map:
      # No ticker holdings on the period beginning, assume zero
      ticker_cf_history_map[ticker] = [{
        "d": period_start_eod,
        "val_eod_before_cf":0.0,
        "cf":0.0,
        "qty_eod_before_cf":0,
        "qty_eod_after_cf":0}]

    txn_eod = txn["ts"].split(" ")[0]
    if txn_eod <= period_start_eod or txn_eod > period_end_eod:
      raise RuntimeError(f"txn dated {txn_eod} is not within period limits {period_start_eod}-{period_end_eod}")

    last_cf_item = ticker_cf_history_map[ticker][-1]
    if last_cf_item["d"] == txn_eod:
      # Recalculate last record
      last_cf_item["qty_eod_after_cf"] += txn["qty"]
      last_cf_item["cf"] += txn["qty"] * txn["price"] # cf uses sale price
    else:
      # Add new record
      ticker_cf_history_map[ticker].append({
        "d": txn_eod,
        "qty_eod_before_cf": last_cf_item["qty_eod_after_cf"],
        "qty_eod_after_cf": last_cf_item["qty_eod_after_cf"] + txn["qty"],
        "cf": txn["qty"] * txn["price"], # cf uses sale price
        "val_eod_before_cf": (last_cf_item["qty_eod_before_cf"]) * eod_price_provider.get_price(txn_eod,ticker) # val uses eod price
      })

  # For each cf, add a period end cf record with the official EOD price (if needed)
  for ticker, cf in ticker_cf_history_map.items():
    last_cf_item = ticker_cf_history_map[ticker][-1]
    if last_cf_item["d"] != period_end_eod:
      # Add new record
      ticker_cf_history_map[ticker].append({
        "d": period_end_eod,
        "qty_eod_before_cf": last_cf_item["qty_eod_after_cf"],
        "qty_eod_after_cf": last_cf_item["qty_eod_after_cf"],
        "cf": 0.0,
        "val_eod_before_cf": last_cf_item["qty_eod_after_cf"] * eod_price_provider.get_price(period_end_eod, ticker) # val uses eod price
      })

  return ticker_cf_history_map

def ticker_cashflows_to_sector_cashflows(sector_ticker_set, ticker_cf_history_map, eod_price_provider):
  sector_cashflows = []

  # Collect all cashflow dates (for active traders,
  # chances are it will be all days except holidays)
  cashflow_dates = set()
  ticker_cf_map_map = {} # List of cf items to map for faster access
  for ticker, ticker_cf in ticker_cf_history_map.items():
    if ticker not in sector_ticker_set:
      continue
    ticker_cf_map_map[ticker] = {}
    
    for ticker_cf_item in ticker_cf:
      cashflow_dates.add(ticker_cf_item["d"])
      ticker_cf_map_map[ticker][ticker_cf_item["d"]] = {
        "cf":ticker_cf_item["cf"],
        "qty_eod_before_cf":ticker_cf_item["qty_eod_before_cf"]
      }

  cur_qty_before_cf_map = {k:0 for k in ticker_cf_map_map.keys()}
  for d in sorted(list(cashflow_dates)):
    for ticker in ticker_cf_map_map.keys():
      if d in ticker_cf_map_map[ticker]:
        cur_qty_before_cf_map[ticker] = ticker_cf_map_map[ticker][d]["qty_eod_before_cf"]
    total_val_eod_before_cf = 0
    total_cf = 0
    for ticker, qty in cur_qty_before_cf_map.items():
      total_val_eod_before_cf += qty * eod_price_provider.get_price(d, ticker)
      if d in ticker_cf_map_map[ticker]:
        total_cf += ticker_cf_map_map[ticker][d]["cf"]

    sector_cashflows.append({"d":d,"val_eod_before_cf":total_val_eod_before_cf,"cf":total_cf})
  return sector_cashflows

def all_sector_cashflows(sector_info_provider, ticker_cf_history, eod_price_provider):
  result = {}
  for sector_tag in sector_info_provider.get_sectors():
    result[sector_tag] = ticker_cashflows_to_sector_cashflows(sector_info_provider.get_sector_tickers(sector_tag), ticker_cf_history, eod_price_provider)
  return result

def twr_cagr(cf_history):
  prev_cf_item = None
  twr = 1.0
  for cf_item in cf_history:
    hpr = 0.0
    if prev_cf_item:
      prev_va_eod_after_cf = prev_cf_item["val_eod_before_cf"]+prev_cf_item["cf"]
      # Do not calc hpr if: cur val is zero (it means no holdings, so hpris zero), or prev val after cf is zero (zero divisor)
      if abs(cf_item["val_eod_before_cf"]) > 0.000001 and abs(prev_va_eod_after_cf) > 0.000000001:
        hpr = cf_item["val_eod_before_cf"] / prev_va_eod_after_cf - 1
      twr = twr * (1+hpr)
    prev_cf_item = cf_item
  d1 = datetime.datetime.strptime(cf_history[0]["d"], "%Y-%m-%d").date()
  d2 = datetime.datetime.strptime(cf_history[-1]["d"], "%Y-%m-%d").date()
  years = (d2-d1).days/365
  cagr = pow(twr, 1/years) - 1.0
  return twr-1, cagr

def txns_and_holdings_to_twr_cagr_by_sector(period_start_eod, period_end_eod, period_start_holdings_json, period_txns_json, eod_price_provider, sector_info_provider):
  ticker_cf_history = txns_and_holdings_to_ticker_cashflows(period_start_eod,period_end_eod, json.loads(period_start_holdings_json), json.loads(period_txns_json),eod_price_provider)
  sector_cf_history_map = all_sector_cashflows(sector_info_provider, ticker_cf_history, eod_price_provider)
  sector_perf_map = {}
  for sector, sector_cf_history in sector_cf_history_map.items():
    twr, cagr = twr_cagr(sector_cf_history)
    sector_perf_map[sector] = {"twr": round(twr,4), "cagr": round(cagr,4)}
  return json.dumps(sector_perf_map,sort_keys=True)

