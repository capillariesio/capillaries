# portfolio_quicktest script and data
## Input files
### Accounts
| account_id | earliest_period_start |
| --- | --- |
| ARKK | 2020-12-31 |
| ARKW | 2020-12-31 |
| ARKF | 2020-12-31 |
| ARKQ | 2020-12-31 |
| ARKG | 2020-12-31 |
| ARKX | 2020-12-31 |
Total 6 rows
### Transactions
| ts | account_id | ticker | qty | price |
| --- | --- | --- | --- | --- |
| 2020-10-16 | ARKK | TSLA | 2466031 | 439.67 |
| 2020-10-20 | ARKK | TSLA | 18992 | 421.94 |
| 2020-10-21 | ARKK | TSLA | 2374 | 422.64 |
| 2020-10-22 | ARKK | TSLA | 7122 | 425.79 |
| 2020-10-23 | ARKK | TSLA | 7122 | 420.63 |
| 2020-10-27 | ARKK | TSLA | 1187 | 424.68 |
| 2020-10-28 | ARKK | TSLA | 4748 | 406.02 |
| 2020-10-29 | ARKK | TSLA | 14244 | 410.83 |
| 2020-11-02 | ARKK | TSLA | 3561 | 400.51 |
| 2020-11-03 | ARKK | TSLA | -151615 | 423.9 |
Total 88459 rows
### Holdings
| account_id | d | ticker | qty |
| --- | --- | --- | --- |
| ARKK | 2020-09-30 | TSLA | 0 |
| ARKK | 2020-12-31 | TSLA | 2660439 |
| ARKK | 2021-03-31 | TSLA | 3757949 |
| ARKK | 2021-06-30 | TSLA | 3566520 |
| ARKK | 2021-09-30 | TSLA | 2545118 |
| ARKK | 2021-12-31 | TSLA | 1198876 |
| ARKK | 2022-03-31 | TSLA | 1094098 |
| ARKK | 2022-06-30 | TSLA | 1023093 |
| ARKK | 2022-09-30 | TSLA | 2927507 |
| ARKK | 2022-12-31 | TSLA | 0 |
Total 4300 rows
## 1_read_txns
### Script node 1_read_txns:
<pre id="json">{
  "1_read_txns": {
    "type": "file_table",
    "desc": "Load txns from csv",
    "explicit_run_only": true,
    "r": {
      "urls": [
        "{dir_in}/txns.csv"
      ],
      "csv": {
        "hdr_line_idx": 0,
        "first_data_line_idx": 1
      },
      "columns": {
        "col_ts": {
          "csv": {
            "col_hdr": "ts"
          },
          "col_type": "string"
        },
        "col_account_id": {
          "csv": {
            "col_hdr": "account_id"
          },
          "col_type": "string"
        },
        "col_ticker": {
          "csv": {
            "col_hdr": "ticker"
          },
          "col_type": "string"
        },
        "col_qty": {
          "csv": {
            "col_hdr": "qty",
            "col_format": "%d"
          },
          "col_type": "int"
        },
        "col_price": {
          "csv": {
            "col_hdr": "price",
            "col_format": "%f"
          },
          "col_type": "float"
        }
      }
    },
    "w": {
      "name": "txns",
      "having": "w.ts > \"{period_start_eod}\" && w.ts <= \"{period_end_eod}\"",
      "fields": {
        "account_id": {
          "expression": "r.col_account_id",
          "type": "string"
        },
        "ts": {
          "expression": "r.col_ts",
          "type": "string"
        },
        "txn_json": {
          "expression": "strings.ReplaceAll(fmt.Sprintf(`{'ts':'%s','t':'%s','q':%d,'p':%s}`, r.col_ts, r.col_ticker, r.col_qty, decimal2(r.col_price)), `'`,`\"`)",
          "type": "string"
        }
      },
      "indexes": {
        "idx_txns_account_id": "non_unique(account_id)"
      }
    }
  }
}</pre>
### Script node 1_read_txns produces Cassandra table txns:
| rowid | account_id | batch_idx | ts | txn_json |
| --- | --- | --- | --- | --- |
| 6227718984371972525 | ARKG | 0 | 2021-07-15 | {"ts":"2021-07-15","t":"CDNA","q":-150785,"p":78.07} |
| 1423528861413970761 | ARKK | 0 | 2022-04-22 | {"ts":"2022-04-22","t":"BEAM","q":32034,"p":42.14} |
| 3744934195534553036 | ARKK | 0 | 2021-11-09 | {"ts":"2021-11-09","t":"SKLZ","q":-26172,"p":12.4} |
| 1000403216417687386 | ARKK | 0 | 2022-04-07 | {"ts":"2022-04-07","t":"PATH","q":-122016,"p":21.08} |
| 2716750518474018218 | ARKG | 0 | 2021-11-02 | {"ts":"2021-11-02","t":"SDGR","q":10161,"p":57.81} |
| 454975264206758237 | ARKG | 0 | 2022-08-11 | {"ts":"2022-08-11","t":"TDOC","q":23847,"p":37.99} |
| 4326447202755688980 | ARKW | 0 | 2021-11-18 | {"ts":"2021-11-18","t":"U","q":-33235,"p":201.12} |
| 3125660748182048369 | ARKK | 0 | 2021-06-08 | {"ts":"2021-06-08","t":"IOVA","q":-14715,"p":20.59} |
| 4910992510239148429 | ARKK | 0 | 2022-11-15 | {"ts":"2022-11-15","t":"TXG","q":29489,"p":40.5} |
| 6092547056903797442 | ARKK | 0 | 2021-07-15 | {"ts":"2021-07-15","t":"NVTA","q":-116655,"p":28.11} |
Total 79487 rows
## 1_read_accounts
### Script node 1_read_accounts:
<pre id="json">{
  "1_read_accounts": {
    "type": "file_table",
    "desc": "Load accounts from csv",
    "explicit_run_only": true,
    "r": {
      "urls": [
        "{dir_in}/accounts.csv"
      ],
      "csv": {
        "hdr_line_idx": 0,
        "first_data_line_idx": 1
      },
      "columns": {
        "col_account_id": {
          "csv": {
            "col_hdr": "account_id"
          },
          "col_type": "string"
        },
        "col_earliest_period_start": {
          "csv": {
            "col_hdr": "earliest_period_start"
          },
          "col_type": "string"
        }
      }
    },
    "w": {
      "name": "accounts",
      "having": "w.earliest_period_start <= \"{period_start_eod}\"",
      "fields": {
        "account_id": {
          "expression": "r.col_account_id",
          "type": "string"
        },
        "earliest_period_start": {
          "expression": "r.col_earliest_period_start",
          "type": "string"
        }
      }
    }
  }
}</pre>
### Script node 1_read_accounts produces Cassandra table accounts:
| rowid | account_id | batch_idx | earliest_period_start |
| --- | --- | --- | --- |
| 748401470040242660 | ARKK | 0 | 2020-12-31 |
| 6520205207003275793 | ARKX | 0 | 2020-12-31 |
| 6621936461880082804 | ARKF | 0 | 2020-12-31 |
| 1522209384545663127 | ARKW | 0 | 2020-12-31 |
| 2380715082357295080 | ARKG | 0 | 2020-12-31 |
| 5559895917703115577 | ARKQ | 0 | 2020-12-31 |
Total 6 rows
## 1_read_period_holdings
### Script node 1_read_period_holdings:
<pre id="json">{
  "1_read_period_holdings": {
    "type": "file_table",
    "desc": "Load holdings from csv",
    "explicit_run_only": true,
    "r": {
      "urls": [
        "{dir_in}/holdings.csv"
      ],
      "csv": {
        "hdr_line_idx": 0,
        "first_data_line_idx": 1
      },
      "columns": {
        "col_eod": {
          "csv": {
            "col_hdr": "d"
          },
          "col_type": "string"
        },
        "col_account_id": {
          "csv": {
            "col_hdr": "account_id"
          },
          "col_type": "string"
        },
        "col_ticker": {
          "csv": {
            "col_hdr": "ticker"
          },
          "col_type": "string"
        },
        "col_qty": {
          "csv": {
            "col_hdr": "qty",
            "col_format": "%d"
          },
          "col_type": "int"
        }
      }
    },
    "w": {
      "name": "period_holdings",
      "having": "\"{period_start_eod}\" <= w.eod && w.eod <= \"{period_end_eod}\"",
      "fields": {
        "account_id": {
          "expression": "r.col_account_id",
          "type": "string"
        },
        "eod": {
          "expression": "r.col_eod",
          "type": "string"
        },
        "holding_json": {
          "expression": "fmt.Sprintf(`{\"d\":\"%s\",\"t\":\"%s\",\"q\":%d}`, r.col_eod, r.col_ticker, r.col_qty)",
          "type": "string"
        }
      },
      "indexes": {
        "idx_period_holdings_account_id": "non_unique(account_id)"
      }
    }
  }
}</pre>
### Script node 1_read_period_holdings produces Cassandra table period_holdings:
| rowid | account_id | batch_idx | eod | holding_json |
| --- | --- | --- | --- | --- |
| 8137955616439595852 | ARKW | 0 | 2022-06-30 | {"d":"2022-06-30","t":"DKNG","q":4136462} |
| 2389372614396780152 | ARKK | 0 | 2020-12-31 | {"d":"2020-12-31","t":"TXG","q":0} |
| 2053788883265150697 | ARKW | 0 | 2022-06-30 | {"d":"2022-06-30","t":"SNAP","q":46} |
| 6589615920203769016 | ARKG | 0 | 2021-09-30 | {"d":"2021-09-30","t":"VERV","q":1260293} |
| 5242651915829609969 | ARKX | 0 | 2021-06-30 | {"d":"2021-06-30","t":"HEI","q":44790} |
| 6128096863450802598 | ARKG | 0 | 2022-03-31 | {"d":"2022-03-31","t":"HIMS","q":176557} |
| 5443763339200360319 | ARKW | 0 | 2021-09-30 | {"d":"2021-09-30","t":"CND","q":2799532} |
| 2796362308442163021 | ARKW | 0 | 2021-09-30 | {"d":"2021-09-30","t":"DKNG UW","q":0} |
| 8726306423823332418 | ARKG | 0 | 2020-12-31 | {"d":"2020-12-31","t":"SLGCW","q":0} |
| 3069581952015718783 | ARKG | 0 | 2021-12-31 | {"d":"2021-12-31","t":"PFE","q":936023} |
Total 3914 rows
## 2_account_txns_outer
### Script node 2_account_txns_outer:
<pre id="json">{
  "2_account_txns_outer": {
    "type": "table_lookup_table",
    "desc": "For each account, merge all txns into single json string",
    "r": {
      "table": "accounts",
      "expected_batches_total": 10
    },
    "l": {
      "index_name": "idx_txns_account_id",
      "join_on": "r.account_id",
      "group": true,
      "join_type": "left"
    },
    "w": {
      "name": "account_txns",
      "fields": {
        "account_id": {
          "expression": "r.account_id",
          "type": "string"
        },
        "txns_json": {
          "expression": "string_agg(l.txn_json,\",\")",
          "type": "string"
        }
      }
    }
  }
}</pre>
### Script node 2_account_txns_outer produces Cassandra table account_txns:
| rowid | account_id | batch_idx | txns_json |
| --- | --- | --- | --- |
| 3719903644314909584 | ARKF | 5 | {"ts":"2022-02-04","t":"Z","q":-2835,"p":48.94},{"ts":"2022-12-31","t":"IPOB","q":-872035,"p":29.5}, ... total length 481037 |
| 7337640464596222089 | ARKX | 6 | {"ts":"2021-04-23","t":"AIR","q":-286,"p":119.13},{"ts":"2021-07-20","t":"AVAV","q":8887,"p":96.85}, ... total length 181595 |
| 4577965328217475423 | ARKK | 0 | {"ts":"2021-03-04","t":"SQ","q":266809,"p":218.41},{"ts":"2021-03-11","t":"SQ","q":120492,"p":241.72 ... total length 1011871 |
| 7773052335557492403 | ARKG | 5 | {"ts":"2022-05-10","t":"PSNL","q":86800,"p":4.4},{"ts":"2022-01-21","t":"IOVA","q":-15248,"p":13.96} ... total length 1084244 |
| 2948673979393982941 | ARKW | 5 | {"ts":"2021-05-28","t":"NFLX","q":-465,"p":502.81},{"ts":"2021-04-30","t":"PYPL","q":-34800,"p":262. ... total length 714787 |
| 6410928427592534468 | ARKQ | 3 | {"ts":"2022-06-15","t":"AVAV","q":-38717,"p":83.61},{"ts":"2022-05-11","t":"SNPS","q":-117,"p":260.8 ... total length 563105 |
Total 6 rows
## 2_account_period_holdings_outer
### Script node 2_account_period_holdings_outer:
<pre id="json">{
  "2_account_period_holdings_outer": {
    "type": "table_lookup_table",
    "desc": "For each account, merge all holdings into single json string",
    "r": {
      "table": "accounts",
      "expected_batches_total": 10
    },
    "l": {
      "index_name": "idx_period_holdings_account_id",
      "join_on": "r.account_id",
      "group": true,
      "join_type": "left"
    },
    "w": {
      "name": "account_period_holdings",
      "fields": {
        "account_id": {
          "expression": "r.account_id",
          "type": "string"
        },
        "holdings_json": {
          "expression": "string_agg(l.holding_json,\",\")",
          "type": "string"
        }
      },
      "indexes": {
        "idx_account_period_holdings_account_id": "unique(account_id)"
      }
    }
  }
}</pre>
### Script node 2_account_period_holdings_outer produces Cassandra table account_period_holdings:
| rowid | account_id | batch_idx | holdings_json |
| --- | --- | --- | --- |
| 2297870030956086858 | ARKK | 0 | {"d":"2022-12-31","t":"PATH","q":0},{"d":"2022-12-31","t":"MTLS","q":0},{"d":"2021-03-31","t":"COIN" ... total length 28246 |
| 3798263933557717806 | ARKF | 5 | {"d":"2022-06-30","t":"ROKU","q":0},{"d":"2022-06-30","t":"BABA","q":200},{"d":"2021-06-30","t":"HDB ... total length 25305 |
| 5291018643877674808 | ARKQ | 3 | {"d":"2021-12-31","t":"ROK","q":1248},{"d":"2022-12-31","t":"WKHS","q":0},{"d":"2022-06-30","t":"ISR ... total length 23268 |
| 6490459023829990771 | ARKX | 6 | {"d":"2021-06-30","t":"XLNX","q":17088},{"d":"2022-03-31","t":"GOOG","q":2899},{"d":"2021-06-30","t" ... total length 17095 |
| 3073048863807356155 | ARKG | 5 | {"d":"2022-03-31","t":"CLLS","q":606},{"d":"2021-03-31","t":"PRME","q":0},{"d":"2020-12-31","t":"PHR ... total length 30734 |
| 7126461419439144862 | ARKW | 5 | {"d":"2022-03-31","t":"FTCH","q":101},{"d":"2021-06-30","t":"JD","q":1452974},{"d":"2022-03-31","t": ... total length 29139 |
Total 6 rows
## 3_build_account_period_activity
### Script node 3_build_account_period_activity:
<pre id="json">{
  "3_build_account_period_activity": {
    "type": "table_lookup_table",
    "desc": "For each account, merge holdings and txns",
    "r": {
      "table": "account_txns",
      "expected_batches_total": 10
    },
    "l": {
      "index_name": "idx_account_period_holdings_account_id",
      "join_on": "r.account_id",
      "group": false,
      "join_type": "left"
    },
    "w": {
      "name": "account_period_activity",
      "fields": {
        "account_id": {
          "expression": "r.account_id",
          "type": "string"
        },
        "txns_json": {
          "expression": " \"[\" + r.txns_json + \"]\" ",
          "type": "string"
        },
        "holdings_json": {
          "expression": " \"[\" + l.holdings_json + \"]\" ",
          "type": "string"
        }
      }
    }
  }
}</pre>
### Script node 3_build_account_period_activity produces Cassandra table account_period_activity:
| rowid | account_id | batch_idx | holdings_json | txns_json |
| --- | --- | --- | --- | --- |
| 2832510643465443886 | ARKW | 6 | [{"d":"2022-03-31","t":"FTCH","q":101},{"d":"2021-06-30","t":"JD","q":1452974},{"d":"2022-03-31","t" ... total length 29141 | [{"ts":"2021-05-28","t":"NFLX","q":-465,"p":502.81},{"ts":"2021-04-30","t":"PYPL","q":-34800,"p":262 ... total length 714789 |
| 3797016840160679147 | ARKX | 1 | [{"d":"2021-06-30","t":"XLNX","q":17088},{"d":"2022-03-31","t":"GOOG","q":2899},{"d":"2021-06-30","t ... total length 17097 | [{"ts":"2021-04-23","t":"AIR","q":-286,"p":119.13},{"ts":"2021-07-20","t":"AVAV","q":8887,"p":96.85} ... total length 181597 |
| 8724188995369700700 | ARKQ | 9 | [{"d":"2021-12-31","t":"ROK","q":1248},{"d":"2022-12-31","t":"WKHS","q":0},{"d":"2022-06-30","t":"IS ... total length 23270 | [{"ts":"2022-06-15","t":"AVAV","q":-38717,"p":83.61},{"ts":"2022-05-11","t":"SNPS","q":-117,"p":260. ... total length 563107 |
| 6149240839756633886 | ARKK | 4 | [{"d":"2022-12-31","t":"PATH","q":0},{"d":"2022-12-31","t":"MTLS","q":0},{"d":"2021-03-31","t":"COIN ... total length 28248 | [{"ts":"2021-03-04","t":"SQ","q":266809,"p":218.41},{"ts":"2021-03-11","t":"SQ","q":120492,"p":241.7 ... total length 1011873 |
| 1335953350070866378 | ARKF | 2 | [{"d":"2022-06-30","t":"ROKU","q":0},{"d":"2022-06-30","t":"BABA","q":200},{"d":"2021-06-30","t":"HD ... total length 25307 | [{"ts":"2022-02-04","t":"Z","q":-2835,"p":48.94},{"ts":"2022-12-31","t":"IPOB","q":-872035,"p":29.5} ... total length 481039 |
| 1945457872909272069 | ARKG | 8 | [{"d":"2022-03-31","t":"CLLS","q":606},{"d":"2021-03-31","t":"PRME","q":0},{"d":"2020-12-31","t":"PH ... total length 30736 | [{"ts":"2022-05-10","t":"PSNL","q":86800,"p":4.4},{"ts":"2022-01-21","t":"IOVA","q":-15248,"p":13.96 ... total length 1084246 |
Total 6 rows
## 4_calc_account_period_perf
### Script node 4_calc_account_period_perf:
<pre id="json">{
  "4_calc_account_period_perf": {
    "type": "table_custom_tfm_table",
    "custom_proc_type": "py_calc",
    "desc": "Apply Python-based calculations to account holdings and txns",
    "r": {
      "table": "account_period_activity",
      "expected_batches_total": 10
    },
    "p": {
      "python_code_urls": [
        "{dir_py}/portfolio_test_company_info_provider.py",
        "{dir_py}/portfolio_test_eod_price_provider.py",
        "{dir_py}/portfolio_calc.py"
      ],
      "calculated_fields": {
        "perf_json": {
          "expression": "txns_and_holdings_to_twr_cagr_by_sector_year_quarter_json(\"{period_start_eod}\", \"{period_end_eod}\", r.holdings_json, r.txns_json, PortfolioTestEodPriceProvider, PortfolioTestCompanyInfoProvider)",
          "type": "string"
        }
      }
    },
    "w": {
      "name": "account_period_perf",
      "fields": {
        "account_id": {
          "expression": "r.account_id",
          "type": "string"
        },
        "perf_json": {
          "expression": "p.perf_json",
          "type": "string"
        }
      }
    }
  }
}</pre>
### Script node 4_calc_account_period_perf produces Cassandra table account_period_perf:
| rowid | account_id | batch_idx | perf_json |
| --- | --- | --- | --- |
| 4248855088935145245 | ARKQ | 9 | {"2021": {"All": {"cagr": -0.0152, "twr": -0.0152}, "Communication Services": {"cagr": 0.062, "twr": ... total length 4674 |
| 6887364451659257268 | ARKF | 6 | {"2021": {"All": {"cagr": -0.1912, "twr": -0.1912}, "Communication Services": {"cagr": -0.4067, "twr ... total length 4679 |
| 5754198081589614465 | ARKK | 6 | {"2021": {"All": {"cagr": -0.2398, "twr": -0.2398}, "Communication Services": {"cagr": -0.3183, "twr ... total length 4731 |
| 32049402890064872 | ARKW | 3 | {"2021": {"All": {"cagr": -0.1949, "twr": -0.1949}, "Communication Services": {"cagr": -0.293, "twr" ... total length 4758 |
| 8662263145276984767 | ARKX | 9 | {"2021": {"All": {"cagr": -0.1055, "twr": -0.1055}, "Communication Services": {"cagr": 0.1719, "twr" ... total length 4544 |
| 1210483319197080237 | ARKG | 5 | {"2021": {"All": {"cagr": -0.3384, "twr": -0.3384}, "Communication Services": {"cagr": 0.1666, "twr" ... total length 4533 |
Total 6 rows
## 5_tag_by_period
### Script node 5_tag_by_period:
<pre id="json">{
  "5_tag_by_period": {
    "type": "table_custom_tfm_table",
    "custom_proc_type": "tag_and_denormalize",
    "desc": "Tag accounts by period name",
    "r": {
      "table": "account_period_perf",
      "expected_batches_total": 10
    },
    "p": {
      "tag_field_name": "period",
      "tag_criteria": {
        "2021": "re.MatchString(`\"2021\":`, r.perf_json)",
        "2021Q1": "re.MatchString(`\"2021Q1\":`, r.perf_json)",
        "2021Q2": "re.MatchString(`\"2021Q2\":`, r.perf_json)",
        "2021Q3": "re.MatchString(`\"2021Q3\":`, r.perf_json)",
        "2021Q4": "re.MatchString(`\"2021Q4\":`, r.perf_json)",
        "2022": "re.MatchString(`\"2022\":`, r.perf_json)",
        "2022Q1": "re.MatchString(`\"2022Q1\":`, r.perf_json)",
        "2022Q2": "re.MatchString(`\"2022Q2\":`, r.perf_json)",
        "2022Q3": "re.MatchString(`\"2022Q3\":`, r.perf_json)",
        "2022Q4": "re.MatchString(`\"2022Q4\":`, r.perf_json)"
      }
    },
    "w": {
      "name": "account_period_perf_by_period",
      "fields": {
        "period": {
          "expression": "p.period",
          "type": "string"
        },
        "account_id": {
          "expression": "r.account_id",
          "type": "string"
        },
        "perf_json": {
          "expression": "r.perf_json",
          "type": "string"
        }
      }
    }
  }
}</pre>
### Script node 5_tag_by_period produces Cassandra table account_period_perf_by_period:
| rowid | account_id | batch_idx | perf_json | period |
| --- | --- | --- | --- | --- |
| 4939073054866503199 | ARKX | 4 | {"2021": {"All": {"cagr": -0.1055, "twr": -0.1055}, "Communication Services": {"cagr": 0.1719, "twr" ... total length 4544 | 2021 |
| 8519649678683220496 | ARKG | 6 | {"2021": {"All": {"cagr": -0.3384, "twr": -0.3384}, "Communication Services": {"cagr": 0.1666, "twr" ... total length 4533 | 2021Q4 |
| 1011747997712188466 | ARKW | 0 | {"2021": {"All": {"cagr": -0.1949, "twr": -0.1949}, "Communication Services": {"cagr": -0.293, "twr" ... total length 4758 | 2021Q4 |
| 6738115215161291018 | ARKW | 0 | {"2021": {"All": {"cagr": -0.1949, "twr": -0.1949}, "Communication Services": {"cagr": -0.293, "twr" ... total length 4758 | 2022Q2 |
| 7605309852895153944 | ARKG | 6 | {"2021": {"All": {"cagr": -0.3384, "twr": -0.3384}, "Communication Services": {"cagr": 0.1666, "twr" ... total length 4533 | 2021Q1 |
| 3848238608544572988 | ARKW | 0 | {"2021": {"All": {"cagr": -0.1949, "twr": -0.1949}, "Communication Services": {"cagr": -0.293, "twr" ... total length 4758 | 2022Q4 |
| 7276369387634914856 | ARKX | 4 | {"2021": {"All": {"cagr": -0.1055, "twr": -0.1055}, "Communication Services": {"cagr": 0.1719, "twr" ... total length 4544 | 2021Q3 |
| 5775092598070466506 | ARKQ | 2 | {"2021": {"All": {"cagr": -0.0152, "twr": -0.0152}, "Communication Services": {"cagr": 0.062, "twr": ... total length 4674 | 2021Q3 |
| 2034702183728472657 | ARKG | 6 | {"2021": {"All": {"cagr": -0.3384, "twr": -0.3384}, "Communication Services": {"cagr": 0.1666, "twr" ... total length 4533 | 2022Q4 |
| 8566446862988426730 | ARKF | 2 | {"2021": {"All": {"cagr": -0.1912, "twr": -0.1912}, "Communication Services": {"cagr": -0.4067, "twr ... total length 4679 | 2021 |
Total 60 rows
## 5_tag_by_sector
### Script node 5_tag_by_sector:
<pre id="json">{
  "5_tag_by_sector": {
    "type": "table_custom_tfm_table",
    "custom_proc_type": "tag_and_denormalize",
    "desc": "Tag accounts by sector",
    "r": {
      "table": "account_period_perf_by_period",
      "expected_batches_total": 10
    },
    "p": {
      "tag_field_name": "sector",
      "tag_criteria": {
        "All": "re.MatchString(`\"All\":`, r.perf_json)",
        "Communication Services": "re.MatchString(`\"Communication Services\":`, r.perf_json)",
        "Consumer Cyclical": "re.MatchString(`\"Consumer Cyclical\":`, r.perf_json)",
        "Consumer Defensive": "re.MatchString(`\"Consumer Defensive\":`, r.perf_json)",
        "Financial Services": "re.MatchString(`\"Financial Services\":`, r.perf_json)",
        "Healthcare": "re.MatchString(`\"Healthcare\":`, r.perf_json)",
        "Industrials": "re.MatchString(`\"Industrials\":`, r.perf_json)",
        "Real Estate": "re.MatchString(`\"Real Estate\":`, r.perf_json)",
        "Technology": "re.MatchString(`\"Technology\":`, r.perf_json)"
      }
    },
    "w": {
      "name": "account_period_perf_by_period_sector",
      "fields": {
        "period": {
          "expression": "r.period",
          "type": "string"
        },
        "sector": {
          "expression": "p.sector",
          "type": "string"
        },
        "account_id": {
          "expression": "r.account_id",
          "type": "string"
        },
        "perf_json": {
          "expression": "r.perf_json",
          "type": "string"
        }
      }
    }
  }
}</pre>
### Script node 5_tag_by_sector produces Cassandra table account_period_perf_by_period_sector:
| rowid | account_id | batch_idx | perf_json | period | sector |
| --- | --- | --- | --- | --- | --- |
| 38619203659759174 | ARKQ | 2 | {"2021": {"All": {"cagr": -0.0152, "twr": -0.0152}, "Communication Services": {"cagr": 0.062, "twr": ... total length 4674 | 2021Q3 | Communication Services |
| 7318994738748804490 | ARKG | 2 | {"2021": {"All": {"cagr": -0.3384, "twr": -0.3384}, "Communication Services": {"cagr": 0.1666, "twr" ... total length 4533 | 2021Q2 | Industrials |
| 9200875608381829281 | ARKX | 0 | {"2021": {"All": {"cagr": -0.1055, "twr": -0.1055}, "Communication Services": {"cagr": 0.1719, "twr" ... total length 4544 | 2021Q1 | Financial Services |
| 268450640519970859 | ARKW | 9 | {"2021": {"All": {"cagr": -0.1949, "twr": -0.1949}, "Communication Services": {"cagr": -0.293, "twr" ... total length 4758 | 2022Q1 | Real Estate |
| 5930178946587274701 | ARKX | 6 | {"2021": {"All": {"cagr": -0.1055, "twr": -0.1055}, "Communication Services": {"cagr": 0.1719, "twr" ... total length 4544 | 2022 | All |
| 4656821578726468050 | ARKW | 6 | {"2021": {"All": {"cagr": -0.1949, "twr": -0.1949}, "Communication Services": {"cagr": -0.293, "twr" ... total length 4758 | 2021Q2 | Industrials |
| 1085903896076097330 | ARKQ | 5 | {"2021": {"All": {"cagr": -0.0152, "twr": -0.0152}, "Communication Services": {"cagr": 0.062, "twr": ... total length 4674 | 2021Q4 | Healthcare |
| 8192073723473730539 | ARKK | 9 | {"2021": {"All": {"cagr": -0.2398, "twr": -0.2398}, "Communication Services": {"cagr": -0.3183, "twr ... total length 4731 | 2022Q1 | Consumer Defensive |
| 1515597991369295446 | ARKQ | 6 | {"2021": {"All": {"cagr": -0.0152, "twr": -0.0152}, "Communication Services": {"cagr": 0.062, "twr": ... total length 4674 | 2022 | Technology |
| 3894919988393358397 | ARKX | 5 | {"2021": {"All": {"cagr": -0.1055, "twr": -0.1055}, "Communication Services": {"cagr": 0.1719, "twr" ... total length 4544 | 2022Q4 | Industrials |
Total 540 rows
## 6_perf_json_to_columns
### Script node 6_perf_json_to_columns:
<pre id="json">{
  "6_perf_json_to_columns": {
    "type": "table_custom_tfm_table",
    "custom_proc_type": "py_calc",
    "desc": "Use Python to read perf json and save stats as columns",
    "r": {
      "table": "account_period_perf_by_period_sector",
      "expected_batches_total": 100
    },
    "p": {
      "python_code_urls": [
        "{dir_py}/json_to_columns.py"
      ],
      "calculated_fields": {
        "twr": {
          "expression": "json_to_twr(r.perf_json, r.period, r.sector)",
          "type": "float"
        },
        "cagr": {
          "expression": "json_to_cagr(r.perf_json, r.period, r.sector)",
          "type": "float"
        }
      }
    },
    "w": {
      "name": "account_period_sector_twr_cagr",
      "fields": {
        "account_id": {
          "expression": "r.account_id",
          "type": "string"
        },
        "period": {
          "expression": "r.period",
          "type": "string"
        },
        "sector": {
          "expression": "r.sector",
          "type": "string"
        },
        "twr": {
          "expression": "p.twr",
          "type": "float"
        },
        "cagr": {
          "expression": "p.cagr",
          "type": "float"
        }
      }
    }
  }
}</pre>
### Script node 6_perf_json_to_columns produces Cassandra table account_period_sector_twr_cagr:
| rowid | account_id | batch_idx | cagr | period | sector | twr |
| --- | --- | --- | --- | --- | --- | --- |
| 8156319240356832475 | ARKK | 4 | 51.01 | 2021Q4 | Financial Services | 10.95 |
| 1727460285331236735 | ARKX | 54 | -45.18 | 2021Q3 | Technology | -14.06 |
| 605972862298258582 | ARKW | 73 | -68.55 | 2022 | Healthcare | -68.55 |
| 2596130854164512297 | ARKQ | 35 | 38.64 | 2021Q1 | Communication Services | 8.39 |
| 2041122392610635755 | ARKK | 59 | -60.53 | 2022 | Healthcare | -60.53 |
| 6176115494321693396 | ARKG | 7 | -33.84 | 2021 | All | -33.84 |
| 2145515218512601521 | ARKW | 2 | 15.93 | 2021Q3 | Financial Services | 3.8 |
| 3378075740549622748 | ARKK | 35 | -84.64 | 2021Q3 | Industrials | -37.63 |
| 2196384824360440404 | ARKQ | 70 | 125.03 | 2021 | Financial Services | 125.03 |
| 7602879978653798310 | ARKK | 44 | 80.61 | 2021Q2 | Technology | 15.88 |
Total 540 rows
## 7_file_account_period_sector_perf
### Script node 7_file_account_period_sector_perf:
<pre id="json">{
  "7_file_account_period_sector_perf": {
    "type": "table_file",
    "desc": "Write yearly/quarterly perf results by sector to CSV file",
    "r": {
      "table": "account_period_sector_twr_cagr"
    },
    "w": {
      "top": {
        "order": "account_id,period,sector"
      },
      "url_template": "{dir_out}/account_period_sector_perf.csv",
      "columns": [
        {
          "csv": {
            "header": "ARK fund",
            "format": "%s"
          },
          "name": "account_id",
          "expression": "r.account_id",
          "type": "string"
        },
        {
          "csv": {
            "header": "Period",
            "format": "%s"
          },
          "name": "period",
          "expression": "r.period",
          "type": "string"
        },
        {
          "csv": {
            "header": "Sector",
            "format": "%s"
          },
          "name": "sector",
          "expression": "r.sector",
          "type": "string"
        },
        {
          "csv": {
            "header": "Time-weighted annualized return %",
            "format": "%.2f"
          },
          "name": "cagr",
          "expression": "r.cagr",
          "type": "float"
        }
      ]
    }
  }
}</pre>
### Script node 7_file_account_period_sector_perf produces data file:
| ARK fund | Period | Sector | Time-weighted annualized return % |
| --- | --- | --- | --- |
| ARKF | 2021 | All | -19.12 |
| ARKF | 2021 | Communication Services | -40.67 |
| ARKF | 2021 | Consumer Cyclical | -22.28 |
| ARKF | 2021 | Consumer Defensive | 0.00 |
| ARKF | 2021 | Financial Services | 16.53 |
| ARKF | 2021 | Healthcare | -55.78 |
| ARKF | 2021 | Industrials | -14.41 |
| ARKF | 2021 | Real Estate | -44.99 |
| ARKF | 2021 | Technology | -17.33 |
| ARKF | 2021Q1 | All | 176.50 |
Total 540 rows
## 7_file_account_year_perf
### Script node 7_file_account_year_perf:
<pre id="json">{
  "7_file_account_year_perf": {
    "type": "table_file",
    "desc": "Write yearly perf results for all sectors to CSV file",
    "r": {
      "table": "account_period_sector_twr_cagr"
    },
    "w": {
      "top": {
        "order": "account_id,period"
      },
      "having": "len(w.period) == 4 && w.sector == \"All\"",
      "url_template": "{dir_out}/account_year_perf.csv",
      "columns": [
        {
          "csv": {
            "header": "ARK fund",
            "format": "%s"
          },
          "name": "account_id",
          "expression": "r.account_id",
          "type": "string"
        },
        {
          "csv": {
            "header": "Period",
            "format": "%s"
          },
          "name": "period",
          "expression": "r.period",
          "type": "string"
        },
        {
          "csv": {
            "header": "Sector",
            "format": "%s"
          },
          "name": "sector",
          "expression": "r.sector",
          "type": "string"
        },
        {
          "csv": {
            "header": "Time-weighted annualized return %",
            "format": "%.2f"
          },
          "name": "cagr",
          "expression": "r.cagr",
          "type": "float"
        }
      ]
    }
  }
}</pre>
### Script node 7_file_account_year_perf produces data file:
| ARK fund | Period | Sector | Time-weighted annualized return % |
| --- | --- | --- | --- |
| ARKF | 2021 | All | -19.12 |
| ARKF | 2022 | All | -63.35 |
| ARKG | 2021 | All | -33.84 |
| ARKG | 2022 | All | -50.87 |
| ARKK | 2021 | All | -23.98 |
| ARKK | 2022 | All | -68.07 |
| ARKQ | 2021 | All | -1.52 |
| ARKQ | 2022 | All | -61.45 |
| ARKW | 2021 | All | -19.49 |
| ARKW | 2022 | All | -65.28 |
Total 12 rows
</div></body></head>
