from json import loads

# Get value from a 1-level map
def get_amt_by_quarter(q, budget_json):
	return float(loads(budget_json)[q])

# Get value from a 2-level map
def get_amt_by_key_and_quarter(k, q, budget_json):
	if isinstance(k, int):
		k = str(k) # sector ids are integers
	return float(loads(budget_json)[k][q])