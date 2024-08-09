# fannie_mae_quicktest script and data
## Input files
### Payments
| Current Actual UPB | UPB at Issuance | Original Interest Rate | Loan Identifier | Seller Name | Monthly Reporting Period | Remaining Months To Maturity | Origination Date | Deal Name | Borrower Credit Score at Origination | Zero Balance Effective Date | Scheduled Principal Current | Remaining Months to Legal Maturity | Original UPB | Original Loan Term |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 485775.65 | 485775.65 | 6.875 | 136610574 | Flagstar Bank, National Association | 20230820 | 350 | 20221020 | CAS 2023 R08 G1 | 735 | 0 | 0 | 351 | 490000 | 360 |
| 248112.85 | 248112.85 | 6.375 | 136610575 | Planet Home Lending, LLC | 20230820 | 352 | 20221020 | CAS 2023 R08 G1 | 709 | 0 | 0 | 352 | 250000 | 360 |
| 468571.34 | 468571.34 | 6.5 | 136610576 | Rocket Mortgage, LLC | 20230820 | 351 | 20221020 | CAS 2023 R08 G1 | 787 | 0 | 0 | 351 | 473000 | 360 |
| 307534.77 | 307534.77 | 5.25 | 136610577 | Rocket Mortgage, LLC | 20230820 | 351 | 20220920 | CAS 2023 R08 G1 | 732 | 0 | 0 | 351 | 311000 | 360 |
| 348025.67 | 348025.67 | 5.5 | 136610578 | U.S. Bank N.A. | 20230820 | 350 | 20221020 | CAS 2023 R08 G1 | 763 | 0 | 0 | 351 | 352000 | 360 |
| 139258.92 | 139258.92 | 6 | 136610579 | Other | 20230820 | 339 | 20220920 | CAS 2023 R08 G1 | 720 | 0 | 0 | 351 | 142000 | 360 |
| 202982.62 | 202982.62 | 5.875 | 136610580 | U.S. Bank N.A. | 20230820 | 351 | 20221020 | CAS 2023 R08 G1 | 799 | 0 | 0 | 351 | 205000 | 360 |
| 172686.14 | 172686.14 | 6.575 | 136610581 | Planet Home Lending, LLC | 20230820 | 352 | 20221020 | CAS 2023 R08 G1 | 690 | 0 | 0 | 352 | 174000 | 360 |
| 110751.32 | 110751.32 | 5.375 | 136610582 | Other | 20230820 | 323 | 20221020 | CAS 2023 R08 G1 | 762 | 0 | 0 | 351 | 116000 | 360 |
| 231756.02 | 231756.02 | 5.5 | 136610583 | JPMorgan Chase Bank, National Association | 20230820 | 350 | 20220920 | CAS 2023 R08 G1 | 795 | 0 | 0 | 350 | 234000 | 360 |

Total 30200 rows
## 01_read_payments
### Script node 01_read_payments:
<pre id="json">{
  "01_read_payments": {
    "type": "file_table",
    "desc": "Read source files to a table",
    "start_policy": "manual",
    "r": {
      "urls": "{file_urls|stringlist}",
      "columns": {
        "col_loan_id": {
          "col_type": "int",
          "parquet": {
            "col_name": "Loan Identifier"
          }
        },
        "col_deal_name": {
          "col_type": "string",
          "parquet": {
            "col_name": "Deal Name"
          }
        },
        "col_seller_name": {
          "col_type": "string",
          "parquet": {
            "col_name": "Seller Name"
          }
        },
        "col_origination_date": {
          "col_type": "int",
          "parquet": {
            "col_name": "Origination Date"
          }
        },
        "col_original_interest_rate": {
          "col_type": "float",
          "parquet": {
            "col_name": "Original Interest Rate"
          }
        },
        "col_borrower_credit_score_at_origination": {
          "col_type": "int",
          "parquet": {
            "col_name": "Borrower Credit Score at Origination"
          }
        },
        "col_original_upb": {
          "col_type": "decimal2",
          "parquet": {
            "col_name": "Original UPB"
          }
        },
        "col_upb_at_issuance": {
          "col_type": "decimal2",
          "parquet": {
            "col_name": "UPB at Issuance"
          }
        },
        "col_original_loan_term": {
          "col_type": "int",
          "parquet": {
            "col_name": "Original Loan Term"
          }
        },
        "col_monthly_reporting_period": {
          "col_type": "int",
          "parquet": {
            "col_name": "Monthly Reporting Period"
          }
        },
        "col_current_actual_upb": {
          "col_type": "decimal2",
          "parquet": {
            "col_name": "Current Actual UPB"
          }
        },
        "col_remaining_months_to_legal_maturity": {
          "col_type": "int",
          "parquet": {
            "col_name": "Remaining Months to Legal Maturity"
          }
        },
        "col_remaining_months_to_maturity": {
          "col_type": "int",
          "parquet": {
            "col_name": "Remaining Months To Maturity"
          }
        },
        "col_zero_balance_effective_date": {
          "col_type": "int",
          "parquet": {
            "col_name": "Zero Balance Effective Date"
          }
        }
      }
    },
    "w": {
      "name": "payments",
      "having_tmp": "(w.zero_balance_effective_date == 0 || w.scheduled_principal_current > 0) && w.original_income > 0",
      "fields": {
        "loan_id": {
          "expression": "r.col_loan_id",
          "type": "int"
        },
        "deal_name": {
          "expression": "r.col_deal_name",
          "type": "string"
        },
        "origination_date": {
          "expression": "r.col_origination_date",
          "type": "int"
        },
        "seller_name": {
          "expression": "r.col_seller_name",
          "type": "string"
        },
        "original_interest_rate": {
          "expression": "r.col_original_interest_rate",
          "type": "float"
        },
        "borrower_credit_score_at_origination": {
          "expression": "r.col_borrower_credit_score_at_origination",
          "type": "int"
        },
        "original_upb": {
          "expression": "r.col_original_upb",
          "type": "decimal2"
        },
        "upb_at_issuance": {
          "expression": "r.col_upb_at_issuance",
          "type": "decimal2"
        },
        "original_loan_term": {
          "expression": "r.col_original_loan_term",
          "type": "int"
        },
        "payment_json": {
          "expression": "strings.ReplaceAll(fmt.Sprintf(`{'monthly_reporting_period':%d,'current_actual_upb':%s,'remaining_months_to_legal_maturity':%d,'remaining_months_to_maturity':%d,'zero_balance_effective_date':%d}`,r.col_monthly_reporting_period, r.col_current_actual_upb, r.col_remaining_months_to_legal_maturity, r.col_remaining_months_to_maturity,r.col_zero_balance_effective_date), `'`,`\"`)",
          "type": "string"
        }
      },
      "indexes": {
        "idx_payments_by_loan_id": "non_unique(loan_id)"
      }
    }
  }
}</pre>
### Script node 01_read_payments produces Cassandra table payments:
| rowid | batch_idx | borrower_credit_score_at_origination | deal_name | loan_id | original_interest_rate | original_loan_term | original_upb | origination_date | payment_json | seller_name | upb_at_issuance |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 7009154683924029194 | 0 | 772 | CAS 2023 R08 G1 | 136617495 | 6.49 | 360 | 351000.00 | 20221020 | {"monthly_reporting_period":20230820,"current_actual_upb":348075.99,"remaining_months_to_legal_matur ... total length 176 | Fairway Independent Mortgage Corporation | 348075.99 |
| 1404844201780415891 | 0 | 766 | CAS 2023 R08 G1 | 136612593 | 5.5 | 360 | 130000.00 | 20221020 | {"monthly_reporting_period":20230820,"current_actual_upb":128695.6,"remaining_months_to_legal_maturi ... total length 175 | loanDepot.com, LLC | 128695.60 |
| 7675668920519271153 | 1 | 720 | CAS 2023 R08 G1 | 136663102 | 5.375 | 360 | 600000.00 | 20220920 | {"monthly_reporting_period":20230820,"current_actual_upb":593139.56,"remaining_months_to_legal_matur ... total length 176 | Other | 593139.56 |
| 1326957998758862938 | 0 | 800 | CAS 2023 R08 G1 | 136626277 | 7.375 | 360 | 416000.00 | 20221120 | {"monthly_reporting_period":20230820,"current_actual_upb":413332.97,"remaining_months_to_legal_matur ... total length 176 | Other | 413332.97 |
| 5858552684072934531 | 1 | 741 | CAS 2023 R08 G1 | 136667936 | 6.875 | 360 | 209000.00 | 20221020 | {"monthly_reporting_period":20230820,"current_actual_upb":206886.86,"remaining_months_to_legal_matur ... total length 176 | Other | 206886.86 |
| 7512474766405739010 | 1 | 688 | CAS 2023 R08 G1 | 136642347 | 6.5 | 360 | 440000.00 | 20221220 | {"monthly_reporting_period":20230820,"current_actual_upb":437169.96,"remaining_months_to_legal_matur ... total length 176 | Other | 437169.96 |
| 5355391759630843874 | 0 | 793 | CAS 2023 R08 G1 | 136627968 | 6.5 | 360 | 440000.00 | 20221220 | {"monthly_reporting_period":20230820,"current_actual_upb":436756.86,"remaining_months_to_legal_matur ... total length 176 | PHH Mortgage Corporation | 436756.86 |
| 5877699967957280082 | 1 | 747 | CAS 2023 R08 G1 | 136649830 | 4.875 | 360 | 403000.00 | 20220920 | {"monthly_reporting_period":20230820,"current_actual_upb":397953.19,"remaining_months_to_legal_matur ... total length 176 | Wells Fargo Bank, N.A. | 397953.19 |
| 6115358499809336851 | 0 | 772 | CAS 2023 R08 G1 | 136621479 | 5.75 | 360 | 104000.00 | 20220920 | {"monthly_reporting_period":20230820,"current_actual_upb":102890.41,"remaining_months_to_legal_matur ... total length 176 | Fifth Third Bank, National Association | 102890.41 |
| 4449584091962481233 | 1 | 747 | CAS 2023 R08 G1 | 136653320 | 5 | 360 | 93000.00 | 20220920 | {"monthly_reporting_period":20230820,"current_actual_upb":90337.26,"remaining_months_to_legal_maturi ... total length 175 | Other | 90337.26 |

Total 60345 rows
## 02_loan_ids
### Script node 02_loan_ids:
<pre id="json">{
  "02_loan_ids": {
    "type": "distinct_table",
    "desc": "Select distinct loan ids",
    "rerun_policy": "fail",
    "r": {
      "table": "payments",
      "rowset_size": 10000,
      "expected_batches_total": "{expected_batches|number}"
    },
    "w": {
      "name": "loan_ids",
      "fields": {
        "loan_id": {
          "expression": "r.loan_id",
          "type": "int"
        },
        "deal_name": {
          "expression": "r.deal_name",
          "type": "string"
        },
        "origination_date": {
          "expression": "r.origination_date",
          "type": "int"
        },
        "seller_name": {
          "expression": "r.seller_name",
          "type": "string"
        },
        "original_interest_rate": {
          "expression": "r.original_interest_rate",
          "type": "float"
        },
        "borrower_credit_score_at_origination": {
          "expression": "r.borrower_credit_score_at_origination",
          "type": "int"
        },
        "original_upb": {
          "expression": "r.original_upb",
          "type": "decimal2"
        },
        "upb_at_issuance": {
          "expression": "r.upb_at_issuance",
          "type": "decimal2"
        },
        "original_loan_term": {
          "expression": "r.original_loan_term",
          "type": "int"
        }
      },
      "indexes": {
        "idx_loan_ids_loan_id": "unique(loan_id)",
        "idx_loan_ids_deal_name": "non_unique(deal_name)"
      }
    }
  }
}</pre>
### Script node 02_loan_ids produces Cassandra table loan_ids:
| rowid | batch_idx | borrower_credit_score_at_origination | deal_name | loan_id | original_interest_rate | original_loan_term | original_upb | origination_date | seller_name | upb_at_issuance |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 6752608178272806248 | 7 | 701 | CAS 2023 R08 G1 | 136666554 | 5.875 | 360 | 320000.00 | 20220820 | Other | 316322.06 |
| 5266430700445856702 | 4 | 704 | CAS 2023 R08 G1 | 136644734 | 7.625 | 360 | 169000.00 | 20221120 | Other | 167048.07 |
| 6905476334942424028 | 4 | 743 | CAS 2023 R08 G1 | 136646722 | 5.875 | 360 | 244000.00 | 20220920 | Other | 241195.58 |
| 2944990095003651352 | 8 | 787 | CAS 2023 R08 G1 | 136653570 | 4.375 | 360 | 499000.00 | 20221020 | Other | 442394.00 |
| 1093947416060466722 | 5 | 760 | CAS 2023 R08 G1 | 136656080 | 6.875 | 360 | 400000.00 | 20220920 | United Wholesale Mortgage, LLC | 34389.66 |
| 1196091718114697743 | 7 | 745 | CAS 2023 R08 G1 | 136648243 | 6.1 | 360 | 511000.00 | 20221020 | Other | 506455.79 |
| 2534038776161250431 | 5 | 784 | CAS 2023 R08 G1 | 136614592 | 6.625 | 360 | 200000.00 | 20221020 | Other | 198376.39 |
| 2122096588038957732 | 6 | 704 | CAS 2023 R08 G1 | 136621098 | 6.125 | 360 | 420000.00 | 20221020 | Other | 416250.12 |
| 1561582520404290930 | 8 | 671 | CAS 2023 R08 G1 | 136660590 | 5.375 | 360 | 273000.00 | 20220820 | Lakeview Loan Servicing, LLC | 269310.18 |
| 7868265320365713909 | 1 | 715 | CAS 2023 R08 G1 | 136636015 | 7.5 | 360 | 400000.00 | 20221220 | Guild Mortgage Company LLC | 396977.70 |

Total 60345 rows
## 02_deal_names
### Script node 02_deal_names:
<pre id="json">{
  "02_deal_names": {
    "type": "distinct_table",
    "desc": "Select distinct deal names",
    "rerun_policy": "fail",
    "r": {
      "table": "loan_ids",
      "rowset_size": 10000,
      "expected_batches_total": "{expected_batches|number}"
    },
    "w": {
      "name": "deal_names",
      "fields": {
        "deal_name": {
          "expression": "r.deal_name",
          "type": "string"
        }
      },
      "indexes": {
        "idx_deal_names_deal_name": "unique(deal_name)"
      }
    }
  }
}</pre>
### Script node 02_deal_names produces Cassandra table deal_names:
| rowid | batch_idx | deal_name |
| --- | --- | --- |
| 877194347010206401 | 8 | CAS 2023 R08 G1 |

Total 1 rows
## 03_deal_total_upbs
### Script node 03_deal_total_upbs:
<pre id="json">{
  "03_deal_total_upbs": {
    "type": "table_lookup_table",
    "desc": "For each deal, calculate total UPBs",
    "r": {
      "table": "deal_names",
      "expected_batches_total": "{expected_batches|number}"
    },
    "l": {
      "index_name": "idx_loan_ids_deal_name",
      "join_on": "r.deal_name",
      "idx_read_batch_size": 10000,
      "right_lookup_read_batch_size": 10000,
      "group": true,
      "join_type": "left"
    },
    "w": {
      "name": "deal_total_upbs",
      "fields": {
        "deal_name": {
          "expression": "r.deal_name",
          "type": "string"
        },
        "total_original_upb": {
          "expression": "sum(l.original_upb)",
          "type": "decimal2"
        },
        "total_upb_at_issuance": {
          "expression": "sum(l.upb_at_issuance)",
          "type": "decimal2"
        },
        "total_original_upb_for_nonzero_rates": {
          "expression": "sum_if(l.original_upb, l.original_interest_rate > 0)",
          "type": "decimal2"
        },
        "total_original_upb_for_nonzero_credit_scores": {
          "expression": "sum_if(l.original_upb, l.borrower_credit_score_at_origination > 0)",
          "type": "decimal2"
        }
      }
    }
  }
}</pre>
### Script node 03_deal_total_upbs produces Cassandra table deal_total_upbs:
| rowid | batch_idx | deal_name | total_original_upb | total_original_upb_for_nonzero_credit_scores | total_original_upb_for_nonzero_rates | total_upb_at_issuance |
| --- | --- | --- | --- | --- | --- | --- |
| 313607574826253983 | 0 | CAS 2023 R08 G1 | 19356281000.00 | 19339292000.00 | 19356281000.00 | 18880190518.06 |

Total 1 rows
## 04_loan_payment_summaries
### Script node 04_loan_payment_summaries:
<pre id="json">{
  "04_loan_payment_summaries": {
    "type": "table_lookup_table",
    "desc": "For each loan, merge all payments into single json string",
    "r": {
      "table": "loan_ids",
      "expected_batches_total": "{expected_batches|number}"
    },
    "l": {
      "index_name": "idx_payments_by_loan_id",
      "join_on": "r.loan_id",
      "group": true,
      "join_type": "left"
    },
    "w": {
      "name": "loan_payment_summaries",
      "fields": {
        "loan_id": {
          "expression": "r.loan_id",
          "type": "int"
        },
        "deal_name": {
          "expression": "r.deal_name",
          "type": "string"
        },
        "origination_date": {
          "expression": "r.origination_date",
          "type": "int"
        },
        "seller_name": {
          "expression": "r.seller_name",
          "type": "string"
        },
        "original_interest_rate": {
          "expression": "r.original_interest_rate",
          "type": "float"
        },
        "borrower_credit_score_at_origination": {
          "expression": "r.borrower_credit_score_at_origination",
          "type": "int"
        },
        "original_upb": {
          "expression": "r.original_upb",
          "type": "decimal2"
        },
        "upb_at_issuance": {
          "expression": "r.upb_at_issuance",
          "type": "decimal2"
        },
        "original_loan_term": {
          "expression": "r.original_loan_term",
          "type": "int"
        },
        "payments_json": {
          "expression": "string_agg(l.payment_json,\",\")",
          "type": "string"
        }
      }
    }
  }
}</pre>
### Script node 04_loan_payment_summaries produces Cassandra table loan_payment_summaries:
| rowid | batch_idx | borrower_credit_score_at_origination | deal_name | loan_id | original_interest_rate | original_loan_term | original_upb | origination_date | payments_json | seller_name | upb_at_issuance |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 8090133554728515063 | 5 | 730 | CAS 2023 R08 G1 | 136620780 | 6.875 | 360 | 345000.00 | 20221020 | {"monthly_reporting_period":20230820,"current_actual_upb":340953.08,"remaining_months_to_legal_matur ... total length 176 | Planet Home Lending, LLC | 340953.08 |
| 4217324607985691962 | 7 | 792 | CAS 2023 R08 G1 | 136630616 | 6.5 | 360 | 208000.00 | 20221220 | {"monthly_reporting_period":20230820,"current_actual_upb":206466.9,"remaining_months_to_legal_maturi ... total length 175 | Other | 206466.90 |
| 8575149161731351342 | 0 | 703 | CAS 2023 R08 G1 | 136670513 | 6.375 | 360 | 124000.00 | 20221020 | {"monthly_reporting_period":20230820,"current_actual_upb":122312.49,"remaining_months_to_legal_matur ... total length 176 | Wells Fargo Bank, N.A. | 122312.49 |
| 6388452409445645625 | 0 | 773 | CAS 2023 R08 G1 | 136631992 | 6.5 | 360 | 492000.00 | 20221120 | {"monthly_reporting_period":20230820,"current_actual_upb":487877.34,"remaining_months_to_legal_matur ... total length 176 | Other | 487877.34 |
| 6370005162485947181 | 2 | 670 | CAS 2023 R08 G1 | 136616160 | 6.625 | 360 | 275000.00 | 20221020 | {"monthly_reporting_period":20230820,"current_actual_upb":272569.19,"remaining_months_to_legal_matur ... total length 176 | Planet Home Lending, LLC | 272569.19 |
| 8164860146446539821 | 8 | 746 | CAS 2023 R08 G1 | 136628157 | 6.375 | 360 | 430000.00 | 20221020 | {"monthly_reporting_period":20230820,"current_actual_upb":305865.81,"remaining_months_to_legal_matur ... total length 176 | Wells Fargo Bank, N.A. | 305865.81 |
| 2802017730424773650 | 6 | 756 | CAS 2023 R08 G1 | 136633888 | 6.5 | 360 | 300000.00 | 20221020 | {"monthly_reporting_period":20230820,"current_actual_upb":297481.33,"remaining_months_to_legal_matur ... total length 176 | NewRez LLC | 297481.33 |
| 7248523927485119366 | 9 | 816 | CAS 2023 R08 G1 | 136647481 | 4.99 | 360 | 188000.00 | 20220920 | {"monthly_reporting_period":20230820,"current_actual_upb":185200.29,"remaining_months_to_legal_matur ... total length 176 | DHI Mortgage Company, Ltd. | 185200.29 |
| 1033769432570046286 | 6 | 814 | CAS 2023 R08 G1 | 136648668 | 5.99 | 360 | 354000.00 | 20221020 | {"monthly_reporting_period":20230820,"current_actual_upb":350262.63,"remaining_months_to_legal_matur ... total length 176 | Fairway Independent Mortgage Corporation | 350262.63 |
| 8390629863600812509 | 6 | 781 | CAS 2023 R08 G1 | 136664833 | 6.375 | 360 | 398000.00 | 20221020 | {"monthly_reporting_period":20230820,"current_actual_upb":394115.27,"remaining_months_to_legal_matur ... total length 176 | Other | 394115.27 |

Total 60345 rows
## 04_loan_summaries_calculated
### Script node 04_loan_summaries_calculated:
<pre id="json">{
  "04_loan_summaries_calculated": {
    "type": "table_custom_tfm_table",
    "custom_proc_type": "py_calc",
    "desc": "Apply Python calculations to loan summaries",
    "r": {
      "table": "loan_payment_summaries",
      "rowset_size": 1000,
      "expected_batches_total": "{expected_batches|number}"
    },
    "p": {
      "python_code_urls": [
        "{dir_cfg}/py/payment_calc.py"
      ],
      "calculated_fields": {
        "sorted_payments_json": {
          "expression": "sorted_payments_json(r.payments_json)",
          "type": "string"
        },
        "payments_behind_ratio": {
          "expression": "payments_behind_ratio(r.payments_json)",
          "type": "float"
        },
        "paid_off_amount": {
          "expression": "paid_off_amount(r.original_upb,r.upb_at_issuance,r.payments_json)",
          "type": "decimal2"
        },
        "paid_off_ratio": {
          "expression": "paid_off_ratio(r.original_upb,r.upb_at_issuance,r.payments_json)",
          "type": "float"
        }
      }
    },
    "w": {
      "name": "loan_summaries_calculated",
      "fields": {
        "loan_id": {
          "expression": "r.loan_id",
          "type": "int"
        },
        "deal_name": {
          "expression": "r.deal_name",
          "type": "string"
        },
        "origination_date": {
          "expression": "r.origination_date",
          "type": "int"
        },
        "seller_name": {
          "expression": "r.seller_name",
          "type": "string"
        },
        "original_interest_rate": {
          "expression": "r.original_interest_rate",
          "type": "float"
        },
        "borrower_credit_score_at_origination": {
          "expression": "r.borrower_credit_score_at_origination",
          "type": "int"
        },
        "original_upb": {
          "expression": "r.original_upb",
          "type": "decimal2"
        },
        "upb_at_issuance": {
          "expression": "r.upb_at_issuance",
          "type": "decimal2"
        },
        "original_loan_term": {
          "expression": "r.original_loan_term",
          "type": "int"
        },
        "is_original_loan_term_30y": {
          "expression": "int.iif(r.original_loan_term == 360, 1, 0)",
          "type": "int"
        },
        "payments_json": {
          "expression": "p.sorted_payments_json",
          "type": "string"
        },
        "payments_behind_ratio": {
          "expression": "p.payments_behind_ratio",
          "type": "float"
        },
        "paid_off_amount": {
          "expression": "decimal2(p.paid_off_amount)",
          "type": "decimal2"
        },
        "paid_off_ratio": {
          "expression": "p.paid_off_ratio",
          "type": "float"
        }
      },
      "indexes": {
        "idx_loan_summaries_calculated_deal_name": "non_unique(deal_name)",
        "idx_loan_summaries_calculated_deal_name_seller_name": "non_unique(deal_name,seller_name)"
      }
    }
  }
}</pre>
### Script node 04_loan_summaries_calculated produces Cassandra table loan_summaries_calculated:
| rowid | batch_idx | borrower_credit_score_at_origination | deal_name | is_original_loan_term_30y | loan_id | original_interest_rate | original_loan_term | original_upb | origination_date | paid_off_amount | paid_off_ratio | payments_behind_ratio | payments_json | seller_name | upb_at_issuance |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 6491088267903773755 | 5 | 765 | CAS 2023 R08 G1 | 1 | 136647259 | 5.5 | 360 | 340000.00 | 20221020 | 3411.39 | 0.0100335 | 0 | [{"monthly_reporting_period": 20230820, "current_actual_upb": 336588.61, "remaining_months_to_legal_ ... total length 187 | Guaranteed Rate, Inc. | 336588.61 |
| 3002985767379285944 | 6 | 681 | CAS 2023 R08 G1 | 1 | 136669188 | 7.125 | 360 | 107000.00 | 20220920 | 1002.91 | 0.0093729906542 | 0 | [{"monthly_reporting_period": 20230820, "current_actual_upb": 105997.09, "remaining_months_to_legal_ ... total length 187 | Wells Fargo Bank, N.A. | 105997.09 |
| 9135042908970700423 | 5 | 810 | CAS 2023 R08 G1 | 1 | 136645991 | 5.99 | 360 | 157000.00 | 20220920 | 2214.79 | 0.0141069426752 | 0 | [{"monthly_reporting_period": 20230820, "current_actual_upb": 154785.21, "remaining_months_to_legal_ ... total length 187 | Other | 154785.21 |
| 6423432104562802155 | 8 | 787 | CAS 2023 R08 G1 | 1 | 136640080 | 6.375 | 360 | 356000.00 | 20221120 | 2883.87 | 0.008100758427 | 0 | [{"monthly_reporting_period": 20230820, "current_actual_upb": 353116.13, "remaining_months_to_legal_ ... total length 187 | Other | 353116.13 |
| 768626453293339972 | 9 | 672 | CAS 2023 R08 G1 | 1 | 136661101 | 5.875 | 360 | 77000.00 | 20220920 | 802.64 | 0.0104238961039 | 0 | [{"monthly_reporting_period": 20230820, "current_actual_upb": 76197.36, "remaining_months_to_legal_m ... total length 186 | Rocket Mortgage, LLC | 76197.36 |
| 732430525825168963 | 2 | 758 | CAS 2023 R08 G1 | 1 | 136668292 | 5.5 | 360 | 184000.00 | 20220920 | 2058.78 | 0.0111890217391 | 0 | [{"monthly_reporting_period": 20230820, "current_actual_upb": 181941.22, "remaining_months_to_legal_ ... total length 187 | Other | 181941.22 |
| 4692838830583694497 | 8 | 771 | CAS 2023 R08 G1 | 1 | 136666685 | 5.125 | 360 | 235000.00 | 20220920 | 2812.67 | 0.0119688085106 | 0 | [{"monthly_reporting_period": 20230820, "current_actual_upb": 232187.33, "remaining_months_to_legal_ ... total length 187 | Wells Fargo Bank, N.A. | 232187.33 |
| 4743445198874990126 | 4 | 800 | CAS 2023 R08 G1 | 1 | 136613067 | 5.125 | 360 | 525000.00 | 20220920 | 6496.92 | 0.0123750857143 | 0 | [{"monthly_reporting_period": 20230820, "current_actual_upb": 518503.08, "remaining_months_to_legal_ ... total length 187 | Guaranteed Rate, Inc. | 518503.08 |
| 2207215628291232774 | 2 | 818 | CAS 2023 R08 G1 | 1 | 136658282 | 5.625 | 360 | 248000.00 | 20220920 | 5769.44 | 0.0232638709677 | 0 | [{"monthly_reporting_period": 20230820, "current_actual_upb": 242230.56, "remaining_months_to_legal_ ... total length 187 | Wells Fargo Bank, N.A. | 242230.56 |
| 6698092111998010659 | 4 | 785 | CAS 2023 R08 G1 | 1 | 136643945 | 6.5 | 360 | 182000.00 | 20221120 | 1341.44 | 0.0073705494505 | 0 | [{"monthly_reporting_period": 20230820, "current_actual_upb": 180658.56, "remaining_months_to_legal_ ... total length 187 | Movement Mortgage, LLC | 180658.56 |

Total 60345 rows
## 05_deal_seller_summaries
### Script node 05_deal_seller_summaries:
<pre id="json">{
  "05_deal_seller_summaries": {
    "type": "table_lookup_table",
    "desc": "For each deal/seller, calculate aggregates from calculated loan summaries",
    "rerun_policy": "fail",
    "r": {
      "table": "deal_sellers",
      "expected_batches_total": "{expected_batches|number}"
    },
    "l": {
      "index_name": "idx_loan_summaries_calculated_deal_name_seller_name",
      "join_on": "r.deal_name,r.seller_name",
      "group": true,
      "join_type": "left"
    },
    "w": {
      "name": "deal_seller_summaries",
      "fields": {
        "deal_name": {
          "expression": "r.deal_name",
          "type": "string"
        },
        "seller_name": {
          "expression": "r.seller_name",
          "type": "string"
        },
        "avg_original_interest_rate": {
          "expression": "avg(l.original_interest_rate)",
          "type": "float"
        },
        "min_original_interest_rate": {
          "expression": "min(l.original_interest_rate)",
          "type": "float"
        },
        "max_original_interest_rate": {
          "expression": "max(l.original_interest_rate)",
          "type": "float"
        },
        "avg_borrower_credit_score_at_origination": {
          "expression": "avg(float(l.borrower_credit_score_at_origination))",
          "type": "float"
        },
        "min_borrower_credit_score_at_origination": {
          "expression": "min(l.borrower_credit_score_at_origination)",
          "type": "int"
        },
        "max_borrower_credit_score_at_origination": {
          "expression": "max(l.borrower_credit_score_at_origination)",
          "type": "int"
        },
        "total_original_upb": {
          "expression": "sum(l.original_upb)",
          "type": "decimal2"
        },
        "total_upb_at_issuance": {
          "expression": "sum(l.upb_at_issuance)",
          "type": "decimal2"
        },
        "total_loans": {
          "expression": "count()",
          "type": "int"
        },
        "total_original_loan_term_30y": {
          "expression": "sum(l.is_original_loan_term_30y)",
          "type": "int"
        },
        "avg_payments_behind_ratio": {
          "expression": "avg(l.payments_behind_ratio)",
          "type": "float"
        },
        "total_paid_off_amount": {
          "expression": "sum(l.paid_off_amount)",
          "type": "decimal2"
        },
        "avg_paid_off_ratio": {
          "expression": "avg(l.paid_off_ratio)",
          "type": "float"
        }
      }
    }
  }
}</pre>
### Script node 05_deal_seller_summaries produces Cassandra table deal_seller_summaries:
| rowid | avg_borrower_credit_score_at_origination | avg_original_interest_rate | avg_paid_off_ratio | avg_payments_behind_ratio | batch_idx | deal_name | max_borrower_credit_score_at_origination | max_original_interest_rate | min_borrower_credit_score_at_origination | min_original_interest_rate | seller_name | total_loans | total_original_loan_term_30y | total_original_upb | total_paid_off_amount | total_upb_at_issuance |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 6066198704004736390 | 761.10300081103 | 5.938240064882 | 0.0253620346696 | 0 | 4 | CAS 2023 R08 G1 | 823 | 7.625 | 0 | 2.99 | Guaranteed Rate, Inc. | 1233 | 1233 | 443720000.00 | 12278921.05 | 431441078.95 |
| 3112795190318877763 | 753.40875 | 6.69331875 | 0.0216680045482 | 0 | 0 | CAS 2023 R08 G1 | 819 | 8.125 | 0 | 3.5 | CrossCountry Mortgage, LLC | 800 | 798 | 237075000.00 | 5715230.05 | 231359769.95 |
| 2893991749081546317 | 757.352984524687 | 6.344100957996 | 0.0252521387433 | 0 | 6 | CAS 2023 R08 G1 | 829 | 7.625 | 0 | 3.75 | Fairway Independent Mortgage Corporation | 1357 | 1355 | 457905000.00 | 12699866.91 | 445205133.09 |
| 8768448610971920534 | 756.492569002123 | 5.7203492569 | 0.0276388077482 | 0 | 7 | CAS 2023 R08 G1 | 823 | 7.625 | 0 | 3.99 | Fifth Third Bank, National Association | 942 | 941 | 285578000.00 | 8575728.03 | 277002271.97 |
| 7235310274837037295 | 753.587826086957 | 6.534476521739 | 0.019365020949 | 0 | 6 | CAS 2023 R08 G1 | 824 | 8.125 | 623 | 3.75 | Flagstar Bank, National Association | 575 | 575 | 194992000.00 | 4062149.92 | 190929850.08 |
| 6071347103838559409 | 760.955708047411 | 6.040384903306 | 0.022706814064 | 0 | 1 | CAS 2023 R08 G1 | 827 | 8 | 621 | 3.5 | PennyMac Corp. | 1603 | 1599 | 568938000.00 | 13549070.57 | 555388929.43 |
| 8254590749681945889 | 751.008688097307 | 5.941403127715 | 0.0186537553046 | 0 | 0 | CAS 2023 R08 G1 | 821 | 8.125 | 620 | 3 | loanDepot.com, LLC | 1151 | 1151 | 388120000.00 | 7360117.11 | 380759882.89 |
| 5357438395202032473 | 750.76983435048 | 6.552690496949 | 0.0171519399367 | 0 | 5 | CAS 2023 R08 G1 | 825 | 8.125 | 0 | 3 | NewRez LLC | 1147 | 1144 | 379065000.00 | 6383953.49 | 372681046.51 |
| 9033258309346024608 | 758.647607934656 | 4.884947491248 | 0.0228911796314 | 0 | 2 | CAS 2023 R08 G1 | 827 | 7.375 | 0 | 3.25 | DHI Mortgage Company, Ltd. | 857 | 857 | 268469000.00 | 6135452.44 | 262333547.56 |
| 2046922479759246362 | 749.176102459613 | 6.095726240722 | 0.0190210740031 | 0 | 6 | CAS 2023 R08 G1 | 829 | 8.125 | 601 | 3.875 | Rocket Mortgage, LLC | 6871 | 6837 | 2205904000.00 | 43306960.59 | 2162597039.41 |

Total 25 rows
## 05_deal_summaries
### Script node 05_deal_summaries:
<pre id="json">{
  "05_deal_summaries": {
    "type": "table_lookup_table",
    "desc": "For each deal, calculate aggregates from calculated loan summaries",
    "r": {
      "table": "deal_total_upbs",
      "expected_batches_total": "{expected_batches|number}"
    },
    "l": {
      "index_name": "idx_loan_summaries_calculated_deal_name",
      "join_on": "r.deal_name",
      "group": true,
      "join_type": "left"
    },
    "w": {
      "name": "deal_summaries",
      "fields": {
        "deal_name": {
          "expression": "r.deal_name",
          "type": "string"
        },
        "wa_original_interest_rate_by_original_upb": {
          "expression": "sum(l.original_interest_rate * float(l.original_upb))",
          "type": "float"
        },
        "min_original_interest_rate": {
          "expression": "min_if(l.original_interest_rate, l.original_interest_rate > 0)",
          "type": "float"
        },
        "max_original_interest_rate": {
          "expression": "max(l.original_interest_rate)",
          "type": "float"
        },
        "wa_borrower_credit_score_at_origination_by_original_upb": {
          "expression": "sum(float(l.borrower_credit_score_at_origination) * float(l.original_upb))",
          "type": "float"
        },
        "min_borrower_credit_score_at_origination": {
          "expression": "min_if(l.borrower_credit_score_at_origination, l.borrower_credit_score_at_origination > 0)",
          "type": "int"
        },
        "max_borrower_credit_score_at_origination": {
          "expression": "max(l.borrower_credit_score_at_origination)",
          "type": "int"
        },
        "total_original_upb": {
          "expression": "r.total_original_upb",
          "type": "decimal2"
        },
        "total_upb_at_issuance": {
          "expression": "r.total_upb_at_issuance",
          "type": "decimal2"
        },
        "total_original_upb_for_nonzero_rates": {
          "expression": "r.total_original_upb_for_nonzero_rates",
          "type": "decimal2"
        },
        "total_original_upb_for_nonzero_credit_scores": {
          "expression": "r.total_original_upb_for_nonzero_credit_scores",
          "type": "decimal2"
        },
        "total_loans": {
          "expression": "count()",
          "type": "int"
        },
        "total_original_loan_term_30y": {
          "expression": "sum(l.is_original_loan_term_30y)",
          "type": "int"
        },
        "avg_payments_behind_ratio": {
          "expression": "avg(l.payments_behind_ratio)",
          "type": "float"
        },
        "total_paid_off_amount": {
          "expression": "sum(l.paid_off_amount)",
          "type": "decimal2"
        },
        "avg_paid_off_ratio": {
          "expression": "avg(l.paid_off_ratio)",
          "type": "float"
        }
      }
    }
  }
}</pre>
### Script node 05_deal_summaries produces Cassandra table deal_summaries:
| rowid | avg_paid_off_ratio | avg_payments_behind_ratio | batch_idx | deal_name | max_borrower_credit_score_at_origination | max_original_interest_rate | min_borrower_credit_score_at_origination | min_original_interest_rate | total_loans | total_original_loan_term_30y | total_original_upb | total_original_upb_for_nonzero_credit_scores | total_original_upb_for_nonzero_rates | total_paid_off_amount | total_upb_at_issuance | wa_borrower_credit_score_at_origination_by_original_upb | wa_original_interest_rate_by_original_upb |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 5697416122093459290 | 0.0231112037748 | 0 | 6 | CAS 2023 R08 G1 | 832 | 8.125 | 600 | 2.5 | 60345 | 60114 | 19356281000.00 | 19339292000.00 | 19356281000.00 | 476091393.37 | 18880190518.06 | 1.4670429761e+13 | 118134224964 |

Total 1 rows
## 04_write_file_loan_summaries_calculated
### Script node 04_write_file_loan_summaries_calculated:
<pre id="json">{
  "04_write_file_loan_summaries_calculated": {
    "type": "table_file",
    "desc": "Write from table to file loan_summaries.parquet",
    "r": {
      "table": "loan_summaries_calculated"
    },
    "w": {
      "top": {
        "order": "loan_id(asc)"
      },
      "url_template": "{dir_out}/loan_summaries_calculated.parquet",
      "columns": [
        {
          "parquet": {
            "column_name": "loan_id"
          },
          "name": "loan_id",
          "expression": "r.loan_id",
          "type": "int"
        },
        {
          "parquet": {
            "column_name": "deal_name"
          },
          "name": "deal_name",
          "expression": "r.deal_name",
          "type": "string"
        },
        {
          "parquet": {
            "column_name": "seller_name"
          },
          "name": "seller_name",
          "expression": "r.seller_name",
          "type": "string"
        },
        {
          "parquet": {
            "column_name": "original_interest_rate"
          },
          "name": "original_interest_rate",
          "expression": "r.original_interest_rate",
          "type": "float"
        },
        {
          "parquet": {
            "column_name": "borrower_credit_score_at_origination"
          },
          "name": "borrower_credit_score_at_origination",
          "expression": "r.borrower_credit_score_at_origination",
          "type": "int"
        },
        {
          "parquet": {
            "column_name": "origination_date"
          },
          "name": "origination_date",
          "expression": "r.origination_date",
          "type": "int"
        },
        {
          "parquet": {
            "column_name": "original_upb"
          },
          "name": "original_upb",
          "expression": "r.original_upb",
          "type": "decimal2"
        },
        {
          "parquet": {
            "column_name": "upb_at_issuance"
          },
          "name": "upb_at_issuance",
          "expression": "r.upb_at_issuance",
          "type": "decimal2"
        },
        {
          "parquet": {
            "column_name": "original_loan_term"
          },
          "name": "original_loan_term",
          "expression": "r.original_loan_term",
          "type": "int"
        },
        {
          "parquet": {
            "column_name": "payments_json"
          },
          "name": "payments_json",
          "expression": "r.payments_json",
          "type": "string"
        },
        {
          "parquet": {
            "column_name": "payments_behind_ratio"
          },
          "name": "payments_behind_ratio",
          "expression": "math.Round(r.payments_behind_ratio*100000)/100000",
          "type": "float"
        },
        {
          "parquet": {
            "column_name": "paid_off_amount"
          },
          "name": "paid_off_amount",
          "expression": "r.paid_off_amount",
          "type": "decimal2"
        },
        {
          "parquet": {
            "column_name": "paid_off_ratio"
          },
          "name": "paid_off_ratio",
          "expression": "math.Round(r.paid_off_ratio*100000)/100000",
          "type": "float"
        }
      ]
    }
  }
}</pre>
### Script node 04_write_file_loan_summaries_calculated produces data file:
| loan_id | deal_name | seller_name | original_interest_rate | borrower_credit_score_at_origination | origination_date | original_upb | upb_at_issuance | original_loan_term | payments_json | payments_behind_ratio | paid_off_amount | paid_off_ratio |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 136610574 | CAS 2023 R08 G1 | Flagstar Bank, National Association | 6.875 | 735 | 20221020 | 490000 | 485775.65 | 360 | [{monthly_reporting_period": 20230820 |  "current_actual_upb": 485775.65 |  "remaining_months_to_legal_maturity": 351 |  "remaining_months_to_maturity": 350 |  "zero_balance_effective_date": 0}]" | 0 | 4224.35 | 0.00862 |
| 136610575 | CAS 2023 R08 G1 | Planet Home Lending, LLC | 6.375 | 709 | 20221020 | 250000 | 248112.85 | 360 | [{monthly_reporting_period": 20230820 |  "current_actual_upb": 248112.85 |  "remaining_months_to_legal_maturity": 352 |  "remaining_months_to_maturity": 352 |  "zero_balance_effective_date": 0}]" | 0 | 1887.15 | 0.00755 |
| 136610576 | CAS 2023 R08 G1 | Rocket Mortgage, LLC | 6.5 | 787 | 20221020 | 473000 | 468571.34 | 360 | [{monthly_reporting_period": 20230820 |  "current_actual_upb": 468571.34 |  "remaining_months_to_legal_maturity": 351 |  "remaining_months_to_maturity": 351 |  "zero_balance_effective_date": 0}]" | 0 | 4428.66 | 0.00936 |
| 136610577 | CAS 2023 R08 G1 | Rocket Mortgage, LLC | 5.25 | 732 | 20220920 | 311000 | 307534.77 | 360 | [{monthly_reporting_period": 20230820 |  "current_actual_upb": 307534.77 |  "remaining_months_to_legal_maturity": 351 |  "remaining_months_to_maturity": 351 |  "zero_balance_effective_date": 0}]" | 0 | 3465.23 | 0.01114 |
| 136610578 | CAS 2023 R08 G1 | U.S. Bank N.A. | 5.5 | 763 | 20221020 | 352000 | 348025.67 | 360 | [{monthly_reporting_period": 20230820 |  "current_actual_upb": 348025.67 |  "remaining_months_to_legal_maturity": 351 |  "remaining_months_to_maturity": 350 |  "zero_balance_effective_date": 0}]" | 0 | 3974.33 | 0.01129 |
| 136610579 | CAS 2023 R08 G1 | Other | 6 | 720 | 20220920 | 142000 | 139258.92 | 360 | [{monthly_reporting_period": 20230820 |  "current_actual_upb": 139258.92 |  "remaining_months_to_legal_maturity": 351 |  "remaining_months_to_maturity": 339 |  "zero_balance_effective_date": 0}]" | 0 | 2741.08 | 0.0193 |
| 136610580 | CAS 2023 R08 G1 | U.S. Bank N.A. | 5.875 | 799 | 20221020 | 205000 | 202982.62 | 360 | [{monthly_reporting_period": 20230820 |  "current_actual_upb": 202982.62 |  "remaining_months_to_legal_maturity": 351 |  "remaining_months_to_maturity": 351 |  "zero_balance_effective_date": 0}]" | 0 | 2017.38 | 0.00984 |
| 136610581 | CAS 2023 R08 G1 | Planet Home Lending, LLC | 6.575 | 690 | 20221020 | 174000 | 172686.14 | 360 | [{monthly_reporting_period": 20230820 |  "current_actual_upb": 172686.14 |  "remaining_months_to_legal_maturity": 352 |  "remaining_months_to_maturity": 352 |  "zero_balance_effective_date": 0}]" | 0 | 1313.86 | 0.00755 |
| 136610582 | CAS 2023 R08 G1 | Other | 5.375 | 762 | 20221020 | 116000 | 110751.32 | 360 | [{monthly_reporting_period": 20230820 |  "current_actual_upb": 110751.32 |  "remaining_months_to_legal_maturity": 351 |  "remaining_months_to_maturity": 323 |  "zero_balance_effective_date": 0}]" | 0 | 5248.68 | 0.04525 |
| 136610583 | CAS 2023 R08 G1 | JPMorgan Chase Bank, National Association | 5.5 | 795 | 20220920 | 234000 | 231756.02 | 360 | [{monthly_reporting_period": 20230820 |  "current_actual_upb": 231756.02 |  "remaining_months_to_legal_maturity": 350 |  "remaining_months_to_maturity": 350 |  "zero_balance_effective_date": 0}]" | 0 | 2243.98 | 0.00959 |

Total 60345 rows
## 05_write_file_deal_seller_summaries
### Script node 05_write_file_deal_seller_summaries:
<pre id="json">{
  "05_write_file_deal_seller_summaries": {
    "type": "table_file",
    "desc": "Write from table to file deal_seller_summaries.parquet",
    "r": {
      "table": "deal_seller_summaries"
    },
    "w": {
      "top": {
        "order": "deal_name(asc),seller_name(asc)"
      },
      "url_template": "{dir_out}/deal_seller_summaries.parquet",
      "columns": [
        {
          "parquet": {
            "column_name": "deal_name"
          },
          "name": "deal_name",
          "expression": "r.deal_name",
          "type": "string"
        },
        {
          "parquet": {
            "column_name": "seller_name"
          },
          "name": "seller_name",
          "expression": "r.seller_name",
          "type": "string"
        },
        {
          "parquet": {
            "column_name": "avg_original_interest_rate"
          },
          "name": "avg_original_interest_rate",
          "expression": "math.Round(r.avg_original_interest_rate*1000)/1000",
          "type": "float"
        },
        {
          "parquet": {
            "column_name": "min_original_interest_rate"
          },
          "name": "min_original_interest_rate",
          "expression": "r.min_original_interest_rate",
          "type": "float"
        },
        {
          "parquet": {
            "column_name": "max_original_interest_rate"
          },
          "name": "max_original_interest_rate",
          "expression": "r.max_original_interest_rate",
          "type": "float"
        },
        {
          "parquet": {
            "column_name": "avg_borrower_credit_score_at_origination"
          },
          "name": "avg_borrower_credit_score_at_origination",
          "expression": "math.Round(r.avg_borrower_credit_score_at_origination*100)/100",
          "type": "float"
        },
        {
          "parquet": {
            "column_name": "min_borrower_credit_score_at_origination"
          },
          "name": "min_borrower_credit_score_at_origination",
          "expression": "r.min_borrower_credit_score_at_origination",
          "type": "int"
        },
        {
          "parquet": {
            "column_name": "max_borrower_credit_score_at_origination"
          },
          "name": "max_borrower_credit_score_at_origination",
          "expression": "r.max_borrower_credit_score_at_origination",
          "type": "int"
        },
        {
          "parquet": {
            "column_name": "total_original_upb"
          },
          "name": "total_original_upb",
          "expression": "r.total_original_upb",
          "type": "decimal2"
        },
        {
          "parquet": {
            "column_name": "total_original_upb_paid_off_ratio"
          },
          "name": "total_original_upb_paid_off_ratio",
          "expression": "math.Round(float(r.total_paid_off_amount)/float(r.total_original_upb)*100000)/100000",
          "type": "float"
        },
        {
          "parquet": {
            "column_name": "total_upb_at_issuance"
          },
          "name": "total_upb_at_issuance",
          "expression": "r.total_upb_at_issuance",
          "type": "decimal2"
        },
        {
          "parquet": {
            "column_name": "total_loans"
          },
          "name": "total_loans",
          "expression": "r.total_loans",
          "type": "int"
        },
        {
          "parquet": {
            "column_name": "total_original_loan_term_30y"
          },
          "name": "total_original_loan_term_30y",
          "expression": "r.total_original_loan_term_30y",
          "type": "int"
        },
        {
          "parquet": {
            "column_name": "avg_payments_behind_ratio"
          },
          "name": "avg_payments_behind_ratio",
          "expression": "math.Round(r.avg_payments_behind_ratio*100000)/100000",
          "type": "float"
        },
        {
          "parquet": {
            "column_name": "total_paid_off_amount"
          },
          "name": "total_paid_off_amount",
          "expression": "r.total_paid_off_amount",
          "type": "decimal2"
        },
        {
          "parquet": {
            "column_name": "avg_paid_off_ratio"
          },
          "name": "avg_paid_off_ratio",
          "expression": "math.Round(r.avg_paid_off_ratio*100000)/100000",
          "type": "float"
        }
      ]
    }
  }
}</pre>
### Script node 05_write_file_deal_seller_summaries produces data file:
| deal_name | seller_name | avg_original_interest_rate | min_original_interest_rate | max_original_interest_rate | avg_borrower_credit_score_at_origination | min_borrower_credit_score_at_origination | max_borrower_credit_score_at_origination | total_original_upb | total_original_upb_paid_off_ratio | total_upb_at_issuance | total_loans | total_original_loan_term_30y | avg_payments_behind_ratio | total_paid_off_amount | avg_paid_off_ratio |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| CAS 2023 R08 G1 | CitiMortgage, Inc. | 5.552 | 3.25 | 7.125 | 765.14 | 624 | 820 | 276412000 | 0.02568 | 269314771.16 | 746 | 745 | 0 | 7097228.84 | 0.02411 |
| CAS 2023 R08 G1 | CrossCountry Mortgage, LLC | 6.693 | 3.5 | 8.125 | 753.41 | 0 | 819 | 237075000 | 0.02411 | 231359769.95 | 800 | 798 | 0 | 5715230.05 | 0.02167 |
| CAS 2023 R08 G1 | DHI Mortgage Company, Ltd. | 4.885 | 3.25 | 7.375 | 758.65 | 0 | 827 | 268469000 | 0.02285 | 262333547.56 | 857 | 857 | 0 | 6135452.44 | 0.02289 |
| CAS 2023 R08 G1 | Fairway Independent Mortgage Corporation | 6.344 | 3.75 | 7.625 | 757.35 | 0 | 829 | 457905000 | 0.02773 | 445205133.09 | 1357 | 1355 | 0 | 12699866.91 | 0.02525 |
| CAS 2023 R08 G1 | Fifth Third Bank, National Association | 5.72 | 3.99 | 7.625 | 756.49 | 0 | 823 | 285578000 | 0.03003 | 277002271.97 | 942 | 941 | 0 | 8575728.03 | 0.02764 |
| CAS 2023 R08 G1 | Flagstar Bank, National Association | 6.534 | 3.75 | 8.125 | 753.59 | 623 | 824 | 194992000 | 0.02083 | 190929850.08 | 575 | 575 | 0 | 4062149.92 | 0.01937 |
| CAS 2023 R08 G1 | Guaranteed Rate, Inc. | 5.938 | 2.99 | 7.625 | 761.1 | 0 | 823 | 443720000 | 0.02767 | 431441078.95 | 1233 | 1233 | 0 | 12278921.05 | 0.02536 |
| CAS 2023 R08 G1 | Guild Mortgage Company LLC | 6.62 | 4.25 | 8 | 760.82 | 0 | 821 | 233925000 | 0.02983 | 226947065.53 | 685 | 685 | 0 | 6977934.47 | 0.02757 |
| CAS 2023 R08 G1 | JPMorgan Chase Bank, National Association | 6.217 | 3.5 | 8.125 | 759.27 | 620 | 823 | 253524000 | 0.0362 | 244345711.66 | 894 | 884 | 0 | 9178398.2 | 0.03168 |
| CAS 2023 R08 G1 | Lakeview Loan Servicing, LLC | 6.008 | 4.25 | 7.625 | 749.66 | 620 | 821 | 231574000 | 0.01804 | 227396791.34 | 619 | 612 | 0 | 4177208.66 | 0.01805 |

Total 25 rows
## 05_write_file_deal_summaries
### Script node 05_write_file_deal_summaries:
<pre id="json">{
  "05_write_file_deal_summaries": {
    "type": "table_file",
    "desc": "Write from table to file deal_summaries.parquet",
    "r": {
      "table": "deal_summaries"
    },
    "w": {
      "top": {
        "order": "deal_name(asc)"
      },
      "url_template": "{dir_out}/deal_summaries.parquet",
      "columns": [
        {
          "parquet": {
            "column_name": "deal_name"
          },
          "name": "deal_name",
          "expression": "r.deal_name",
          "type": "string"
        },
        {
          "parquet": {
            "column_name": "wa_original_interest_rate_by_original_upb"
          },
          "name": "wa_original_interest_rate_by_original_upb",
          "expression": "math.Round(r.wa_original_interest_rate_by_original_upb/r.total_original_upb_for_nonzero_rates*1000)/1000",
          "type": "float"
        },
        {
          "parquet": {
            "column_name": "min_original_interest_rate"
          },
          "name": "min_original_interest_rate",
          "expression": "r.min_original_interest_rate",
          "type": "float"
        },
        {
          "parquet": {
            "column_name": "max_original_interest_rate"
          },
          "name": "max_original_interest_rate",
          "expression": "r.max_original_interest_rate",
          "type": "float"
        },
        {
          "parquet": {
            "column_name": "wa_borrower_credit_score_at_origination_by_original_upb"
          },
          "name": "wa_borrower_credit_score_at_origination_by_original_upb",
          "expression": "math.Round(r.wa_borrower_credit_score_at_origination_by_original_upb/r.total_original_upb_for_nonzero_credit_scores*100)/100",
          "type": "float"
        },
        {
          "parquet": {
            "column_name": "min_borrower_credit_score_at_origination"
          },
          "name": "min_borrower_credit_score_at_origination",
          "expression": "r.min_borrower_credit_score_at_origination",
          "type": "int"
        },
        {
          "parquet": {
            "column_name": "max_borrower_credit_score_at_origination"
          },
          "name": "max_borrower_credit_score_at_origination",
          "expression": "r.max_borrower_credit_score_at_origination",
          "type": "int"
        },
        {
          "parquet": {
            "column_name": "total_original_upb"
          },
          "name": "total_original_upb",
          "expression": "r.total_original_upb",
          "type": "decimal2"
        },
        {
          "parquet": {
            "column_name": "total_original_upb_paid_off_ratio"
          },
          "name": "total_original_upb_paid_off_ratio",
          "expression": "math.Round(float(r.total_paid_off_amount)/float(r.total_original_upb)*100000)/100000",
          "type": "float"
        },
        {
          "parquet": {
            "column_name": "total_upb_at_issuance"
          },
          "name": "total_upb_at_issuance",
          "expression": "r.total_upb_at_issuance",
          "type": "decimal2"
        },
        {
          "parquet": {
            "column_name": "total_loans"
          },
          "name": "total_loans",
          "expression": "r.total_loans",
          "type": "int"
        },
        {
          "parquet": {
            "column_name": "total_original_loan_term_30y"
          },
          "name": "total_original_loan_term_30y",
          "expression": "r.total_original_loan_term_30y",
          "type": "int"
        },
        {
          "parquet": {
            "column_name": "avg_payments_behind_ratio"
          },
          "name": "avg_payments_behind_ratio",
          "expression": "math.Round(r.avg_payments_behind_ratio*100000)/100000",
          "type": "float"
        },
        {
          "parquet": {
            "column_name": "total_paid_off_amount"
          },
          "name": "total_paid_off_amount",
          "expression": "r.total_paid_off_amount",
          "type": "decimal2"
        },
        {
          "parquet": {
            "column_name": "avg_paid_off_ratio"
          },
          "name": "avg_paid_off_ratio",
          "expression": "math.Round(r.avg_paid_off_ratio*100000)/100000",
          "type": "float"
        }
      ]
    }
  }
}</pre>
### Script node 05_write_file_deal_summaries produces data file:
| deal_name | wa_original_interest_rate_by_original_upb | min_original_interest_rate | max_original_interest_rate | wa_borrower_credit_score_at_origination_by_original_upb | min_borrower_credit_score_at_origination | max_borrower_credit_score_at_origination | total_original_upb | total_original_upb_paid_off_ratio | total_upb_at_issuance | total_loans | total_original_loan_term_30y | avg_payments_behind_ratio | total_paid_off_amount | avg_paid_off_ratio |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| CAS 2023 R08 G1 | 6.103 | 2.5 | 8.125 | 758.58 | 600 | 832 | 19356281000 | 0.0246 | 18880190518.06 | 60345 | 60114 | 0 | 476091393.37 | 0.02311 |

Total 1 rows
