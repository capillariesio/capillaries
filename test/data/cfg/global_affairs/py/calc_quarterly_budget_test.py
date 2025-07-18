from datetime import date
from calc_quarterly_budget import Qtr, int2date, daysInQuarters, amt_to_quarterly_budget_json, map_to_quarterly_budget_json

def _test(expected, actual):
    if actual != expected:
        print(expected)
        print(actual)
    else:
        print("OK")

def test_int2date():
    d1 = int2date(20220331)
    _test(date(year=2022,month=3,day=31),d1)

def test_qtr():
    _test(91,Qtr(2020,1).daysInQtr())
    _test(90,Qtr(2021,1).daysInQtr())
    _test(90,Qtr(2022,1).daysInQtr())
    _test(90,Qtr(2023,1).daysInQtr())

    _test(91,Qtr(2023,2).daysInQtr())
    _test(92,Qtr(2023,3).daysInQtr())
    _test(92,Qtr(2023,4).daysInQtr())

def test_daysInQuarters():
    _test([('2020-Q1', 91), ('2020-Q2', 91), ('2020-Q3', 92), ('2020-Q4', 92)],daysInQuarters(date(2020,1,1),date(2020,12,31)))
    _test([('2019-Q4', 1), ('2020-Q1', 91), ('2020-Q2', 91), ('2020-Q3', 92), ('2020-Q4', 92), ('2021-Q1', 1)],daysInQuarters(date(2019,12,31),date(2021,1,1)))
    _test([('2014-Q1', 2)],daysInQuarters(date(2014,3,30),date(2014,3,31)))
    
def test_amt_to_quarterly_budget_json():
    _test(
        '{"2022-Q4": 1.0, "2023-Q1": 90.0, "2023-Q2": 91.0, "2023-Q3": 92.0, "2023-Q4": 91.0}',
        amt_to_quarterly_budget_json(20221231, 20231230, 365.0))

def test_map_to_quarterly_budget_json():
    _test(
        '{"IN": {"2022-Q4": 1.0, "2023-Q1": 90.0, "2023-Q2": 91.0, "2023-Q3": 92.0, "2023-Q4": 91.0}}',
        map_to_quarterly_budget_json(20221231, 20231230, '{"IN": 365.0}'))

test_int2date()
test_qtr()
test_daysInQuarters()
test_amt_to_quarterly_budget_json()
test_map_to_quarterly_budget_json()