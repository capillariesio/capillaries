from calc_order_items_code import increase_by_ten_percent, next_local_monday, day_of_week, is_weekend, divide_by_three

print(increase_by_ten_percent(10.1) == 11.11)
print(divide_by_three(3.12) == 1.04)
print(next_local_monday("2007-03-05T21:08:12.123+02:00") == "2007-03-05T23:59:59.999+02:00")
print(next_local_monday("2007-03-06T21:08:12.123+02:00") == "2007-03-12T23:59:59.999+02:00")
print(day_of_week("2007-03-12T23:59:59.999+02:00") == "Monday")
print(is_weekend("2007-03-12T21:08:12.123+02:00") == False)
print(is_weekend("2007-03-11T21:08:12.123+02:00") == True)
