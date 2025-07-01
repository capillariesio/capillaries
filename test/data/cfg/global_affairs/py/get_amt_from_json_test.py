from get_amt_from_json import get_amt_by_quarter, get_amt_by_key_and_quarter

def _test(expected, actual):
    if actual != expected:
        print(expected)
        print(actual)
    else:
        print("OK")

def test_get_amt_from_json():
    _test(90.0, get_amt_by_quarter('2023-Q1', '{"2022-Q4": 1.0, "2023-Q1": 90.0, "2023-Q2": 91.0, "2023-Q3": 92.0, "2023-Q4": 91.0}'))
    _test(90.0, get_amt_by_key_and_quarter('IN', '2023-Q1', '{"IN": {"2022-Q4": 1.0, "2023-Q1": 90.0, "2023-Q2": 91.0, "2023-Q3": 92.0, "2023-Q4": 91.0}}'))
    _test(90.0, get_amt_by_key_and_quarter(1, '2023-Q1', '{"1": {"2022-Q4": 1.0, "2023-Q1": 90.0, "2023-Q2": 91.0, "2023-Q3": 92.0, "2023-Q4": 91.0}}'))

test_get_amt_from_json()