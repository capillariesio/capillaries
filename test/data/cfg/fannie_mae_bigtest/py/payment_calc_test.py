from payment_calc import payments_behind_ratio,paid_off_amount,paid_off_ratio

def _test(expected, actual):
    if actual != expected:
        print(expected)
        print(actual)
    else:
        print("OK")




higher_upb_at_issuance_json = '{"monthly_reporting_period":20230920,"current_actual_upb":67025.81,"remaining_months_to_legal_maturity":352,"remaining_months_to_maturity":352,"zero_balance_effective_date":0}'
paid_off_json = '{"monthly_reporting_period":20230920,"current_actual_upb":0,"remaining_months_to_legal_maturity":0,"remaining_months_to_maturity":0,"zero_balance_effective_date":20230920}'

def test_payment_calc():
    _test(0.0, payments_behind_ratio(higher_upb_at_issuance_json))
    _test(51.01, round(paid_off_amount(67000.00,67076.82,higher_upb_at_issuance_json), 2))
    _test(0.00076, round(paid_off_ratio(67000.00,67076.82,higher_upb_at_issuance_json), 5))

    _test(0.0, payments_behind_ratio(paid_off_json))
    _test(970000.00, paid_off_amount(970000.00,960123.41,paid_off_json))
    _test(1.0, paid_off_ratio(970000.00,960123.41,paid_off_json))

test_payment_calc()