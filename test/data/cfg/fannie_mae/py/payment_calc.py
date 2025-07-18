# This is not a production-quality code for actual financial calculations. Use for testing purposes only.
 
import json

def sorted_payments_json(payments_json):
  payments = json.loads("["+payments_json+"]")
  return json.dumps(sorted(payments, key=lambda x: x["monthly_reporting_period"], reverse=False))

def payments_behind_ratio(payments_json):
  payments = json.loads("["+payments_json+"]")
  payments_behind = 0
  payments_total = 0
  for p in sorted(payments, key=lambda x: x["monthly_reporting_period"], reverse=False):
    payments_total += 1
    payments_behind += 1 if p["remaining_months_to_maturity"] > p["remaining_months_to_legal_maturity"] else 0
    if p["zero_balance_effective_date"] > 0:
      break
  return float(payments_behind)/float(payments_total)

def paid_off_amount(original_upb, upb_at_issuance, payments_json):
  payments = json.loads("["+payments_json+"]")
  # Analyse the last recorded payment
  p = sorted(payments, key=lambda x: x["monthly_reporting_period"], reverse=True)[0]
  # For some loans, original_upb < upb_at_issuance (principal revisited?)
  upb_to_pay_off = max(original_upb, upb_at_issuance)
  return upb_to_pay_off - p["current_actual_upb"]
  
def paid_off_ratio(original_upb, upb_at_issuance, payments_json):
  payments = json.loads("["+payments_json+"]")
  # Analyse the last recorded payment
  p = sorted(payments, key=lambda x: x["monthly_reporting_period"], reverse=True)[0]
  # For some loans, original_upb < upb_at_issuance (principal revisited?)
  upb_to_pay_off = max(original_upb, upb_at_issuance)
  return (upb_to_pay_off - p["current_actual_upb"])/upb_to_pay_off
