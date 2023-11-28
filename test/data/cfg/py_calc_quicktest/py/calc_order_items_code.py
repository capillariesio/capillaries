from datetime import datetime, timedelta
from math import floor

def increase_by_ten_percent(numeric_val):
    # We explicitly round the answer here as we assume this is the biz logic.
    # If we do not round it here, p.taxed_value will be written to the table as decimal2 (so, it will be rounded),
    # but in further python calculations (divide_by_three etc) it will keep un-rounded, causing discrepancies
    return floor(float(numeric_val) * 1.10 * 100)/100

def divide_by_three(numeric_val):
    # We deliberately do not round this result as we want to see the difference between decimal and float for this example
    return float(numeric_val) / 3

# Expects and returns ISO 8601 "2007-03-05T21:08:12.123+02:00"
# Python supports upt to microseseconds, but Cassandra supports only milliseconds. Millis are our lingua franca. See PythonDatetimeFormat constant.
# Z zone not supported on the golang side, always return +00:00 instead
# Force .000 milliseconds in the result, golang really needs those
def next_local_monday(ts_string):
    dt = datetime.fromisoformat(ts_string)
    weekday = dt.weekday() # Mon 0, Sun 6 - Python uses metric
    weekday_delta = (7 - weekday) % 7
    dt = dt.replace(hour=23, minute=59, second=59, microsecond=999000) + timedelta(days=weekday_delta)
    return dt.isoformat(timespec='milliseconds') 

def day_of_week(ts_string):
    dt = datetime.fromisoformat(ts_string)
    return dt.strftime("%A")

def is_weekend(ts_string):
    dt = datetime.fromisoformat(ts_string)
    weekday = dt.weekday() # Mon 0, Sun 6 
    return weekday == 5 or weekday == 6

