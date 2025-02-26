from datetime import date, timedelta
from json import loads, dumps

class Qtr:
	def __init__(self, year, quarter):
		self.year = year
		self.quarter = quarter

	def fromDate(d):
		return Qtr(d.year, (d.month-1)//3+1)

	def next(self):
		year = self.year
		quarter = self.quarter + 1
		if quarter == 5:
			quarter = 1
			year = year + 1
		return Qtr(year,quarter)
	
	def _firstMonth(self):
		if self.quarter == 1:
			return 1
		elif self.quarter == 2:
			return 4
		elif self.quarter == 3:
			return 7
		else:
			return 10

	def _lastMonth(self):
		if self.quarter == 1:
			return 3
		elif self.quarter == 2:
			return 6
		elif self.quarter == 3:
			return 9
		else:
			return 12
		
	def daysInQtr(self):
		return (self.lastDay() - self.firstDay()).days + 1
	
	def firstDay(self):
		return date(self.year,self._firstMonth(), 1)

	def lastDay(self):
		next_month = date(self.year,self._lastMonth(), 28)  + timedelta(days=4)
		return next_month - timedelta(days=next_month.day)
	
	def __repr__(self):
		return f"{self.year}-Q{self.quarter}"
	
	def __eq__(self, q):
		return self.year == q.year and self.quarter == q.quarter

def int2date(i):
	return date(year=i // 10000, month=i//100 - i // 10000 * 100, day=i - i//100*100)

# Returns a list of (2020-Q1, 90) tuples
def daysInQuarters(d_start, d_end):
	q_first = Qtr.fromDate(d_start)
	q_last = Qtr.fromDate(d_end)
	days_in_qtr = []
	q = q_first
	while True:
		if q == q_first and q == q_last:
			days_in_qtr.append((str(q),(d_end - d_start).days+1))
			break
		elif q == q_first:
			days_in_qtr.append((str(q),(q.lastDay() - d_start).days+1))
		elif q == q_last:
			days_in_qtr.append((str(q),(d_end - q.firstDay()).days+1))
			break
		else:
			days_in_qtr.append((str(q),q.daysInQtr()))
		q = q.next()
	return days_in_qtr

# Split number into quarterly portions
def amt_to_quarterly_budget_json(start_date, end_date, total_amt):
	d_start = int2date(start_date)
	d_end = int2date(end_date)
	if d_start > d_end:
		raise RuntimeError(f"Start date {d_start} is greater than end date {d_end}")
	
	days_in_qtr = daysInQuarters(d_start, d_end)
	total_days = (d_end - d_start).days + 1

	amt_per_day = total_amt / total_days
	value_map = {}
	for t in days_in_qtr:
		value_map[t[0]] = t[1]*amt_per_day
	return dumps(value_map)

# For each number in map, split number into quarterly portions
def map_to_quarterly_budget_json(start_date, end_date, amt_map):
	d_start = int2date(start_date)
	d_end = int2date(end_date)
	if d_start > d_end:
		raise RuntimeError(f"Start date {d_start} is greater than end date {d_end}")
	
	days_in_qtr = daysInQuarters(d_start, d_end)
	total_days = (d_end - d_start).days + 1

	result = {}
	for key, total_amt in loads(amt_map).items():
		amt_per_day = total_amt / total_days
		value_map = {}
		for t in days_in_qtr:
			value_map[t[0]] = t[1]*amt_per_day
		result[key] = value_map
	return dumps(result)
		

