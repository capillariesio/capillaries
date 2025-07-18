import json

def json_to_twr(perf_json, period, sector):
    return json.loads(perf_json)[period][sector]["twr"]*100

def json_to_cagr(perf_json, period, sector):
    return json.loads(perf_json)[period][sector]["cagr"]*100
