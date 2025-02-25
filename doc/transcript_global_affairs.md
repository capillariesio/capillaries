# global_affairs_quicktest script and data
## Input files
### Global Affairs projects financials
| project_id | partner_id | amt | start_date | end_date | country_amt_json | sector_amt_json |
| --- | --- | --- | --- | --- | --- | --- |
| p012277001 | 29979 | 1209600000.0 | 20231214 | 20251231 | {"R-SOUTHSAHARA": 846720000.0, "R-SOUTHAMERICA": 36288000.0, "R-MIDDLEEAST": 84672000.0, "R-ASIA": 2 ... total length 111 | {"12262": 362880000.0, "12263": 217728000.0, "13040": 628992000.0} |
| p010807001 | 29578 | 1000000000.0 | 20220331 | 20470331 | {"IN": 210000000.0, "PH": 210000000.0, "ID": 210000000.0, "ZA": 210000000.0, "DO": 80000000.0, "MK": ... total length 112 | {"23110": 100000000.0, "23210": 900000000.0} |
| p005247001 | 29979 | 930400000.0 | 20201216 | 20221231 | {"R-SOUTHSAHARA": 604760000.0, "R-SOUTHAMERICA": 37216000.0, "R-MIDDLEEAST": 74432000.0, "R-CENTRALA ... total length 140 | {"12262": 279120000.0, "12263": 158168000.0, "13040": 493112000.0} |
| d003827001 | 29979 | 785000000.0 | 20170731 | 20201231 | {"R-AFRICA": 266900000.0, "R-AMERICA": 259050000.0, "R-ASIA": 259050000.0} | {"12262": 235500000.0, "12263": 133450000.0, "13040": 416050000.0} |
| p012183001 | 29636 | 500000000.0 | 20240308 | 20470331 | {"R-WESTINDIES": 50000000.0, "R-NORTHCENTRALAMERICA": 150000000.0, "R-SOUTHAMERICA": 300000000.0} | {"21012": 25000000.0, "23183": 50000000.0, "23210": 70000000.0, "23230": 125000000.0, "23240": 10000 ... total length 229 |
| d002243001 | 29981 | 500000000.0 | 20150917 | 20211231 | {"R-AFRICA": 359000000.0, "R-AMERICA": 28000000.0, "R-ASIA": 113000000.0} | {"12250": 500000000.0} |
| d000514001 | 29979 | 450000000.01 | 20150327 | 20170331 | {"R-AFRICA": 283500000.01, "R-AMERICA": 36000000.0, "R-ASIA": 103500000.0, "R-EUROPE": 27000000.0} | {"12262": 144000000.0, "12263": 81000000.0, "13040": 225000000.0} |
| m013354001 | 29979 | 450000000.0 | 20111209 | 20141231 | {"R-AFRICA": 225000000.0, "R-AMERICA": 49500000.0, "R-ASIA": 126000000.0, "R-EUROPE": 49500000.0} | {"12262": 99000000.0, "12263": 72000000.0, "13040": 279000000.0} |
| m012761001 | 29979 | 450000000.0 | 20080319 | 20110331 | {"R-AFRICA": 283500000.0, "R-AMERICA": 31500000.0, "R-ASIA": 103500000.0, "R-EUROPE": 31500000.0} | {"12262": 99000000.0, "12263": 72000000.0, "13040": 279000000.0} |
| p006992001 | 29981 | 400000000.0 | 20210330 | 20260331 | {"R-AFRICA": 276000000.0, "R-AMERICA": 4000000.0, "R-ASIA": 120000000.0} | {"12250": 336000000.0, "12350": 20000000.0, "13030": 24000000.0, "13040": 20000000.0} |

Total 19 rows
## 1_read_project_financials
### Script node 1_read_project_financials:
<pre id="json">{
  "1_read_project_financials": {
    "type": "file_table",
    "desc": "Read file project_budget.csv to table",
    "start_policy": "manual",
    "r": {
      "urls": [
        "{dir_in}/{harvested_project_financials_file}"
      ],
      "csv": {
        "hdr_line_idx": 0,
        "first_data_line_idx": 1,
        "separator": ","
      },
      "columns": {
        "project_id": {
          "col_type": "string",
          "csv": {
            "col_hdr": "project_id"
          }
        },
        "partner_id": {
          "col_type": "int",
          "csv": {
            "col_hdr": "partner_id",
            "col_format": "%d"
          }
        },
        "amt": {
          "col_type": "float",
          "csv": {
            "col_hdr": "amt",
            "col_format": "%f"
          }
        },
        "start_date": {
          "col_type": "int",
          "csv": {
            "col_hdr": "start_date",
            "col_format": "%d"
          }
        },
        "end_date": {
          "col_type": "int",
          "csv": {
            "col_hdr": "end_date",
            "col_format": "%d"
          }
        },
        "country_amt_json": {
          "col_type": "string",
          "csv": {
            "col_hdr": "country_amt_json"
          }
        },
        "sector_amt_json": {
          "col_type": "string",
          "csv": {
            "col_hdr": "sector_amt_json"
          }
        }
      }
    },
    "w": {
      "name": "project_financials",
      "fields": {
        "project_id": {
          "expression": "r.project_id",
          "type": "string"
        },
        "partner_id": {
          "expression": "r.partner_id",
          "type": "int"
        },
        "amt": {
          "expression": "r.amt",
          "type": "float"
        },
        "start_date": {
          "expression": "r.start_date",
          "type": "int"
        },
        "end_date": {
          "expression": "r.end_date",
          "type": "int"
        },
        "country_amt_json": {
          "expression": "r.country_amt_json",
          "type": "string"
        },
        "sector_amt_json": {
          "expression": "r.sector_amt_json",
          "type": "string"
        }
      }
    }
  }
}</pre>
### Script node 1_read_project_financials produces Cassandra table project_financials:
| rowid | amt | batch_idx | country_amt_json | end_date | partner_id | project_id | sector_amt_json | start_date |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 3450522840444560684 | 500000000 | 0 | {"R-WESTINDIES": 50000000.0, "R-NORTHCENTRALAMERICA": 150000000.0, "R-SOUTHAMERICA": 300000000.0} | 20470331 | 29636 | p012183001 | {"21012": 25000000.0, "23183": 50000000.0, "23210": 70000000.0, "23230": 125000000.0, "23240": 10000 ... total length 229 | 20240308 |
| 5192768312199147756 | 325615484.069999992847 | 0 | {"R-AFRICA": 325615484.07} | 20140313 | 29663 | m013512001 | {"14021": 16280774.2, "14022": 16280774.2, "14031": 16280774.2, "14032": 16280774.2, "14081": 651230 ... total length 342 | 20110208 |
| 5703125669931000884 | 280000000 | 0 | {"R-AFRICA": 140000000.0, "R-ASIA": 140000000.0} | 20250731 | 29820 | p005253001 | {"12240": 224000000.0, "13020": 56000000.0} | 20190514 |
| 359318821027879808 | 265000000 | 0 | {"R-AFRICA": 159000000.0, "R-AMERICA": 39750000.0, "R-ASIA": 53000000.0, "R-OCEANIA": 13250000.0} | 20260630 | 30039 | p006692001 | {"11110": 26500000.0, "11130": 53000000.0, "11220": 132500000.0, "11320": 53000000.0} | 20220331 |
| 7019190532332255316 | 450000000.009999990463 | 0 | {"R-AFRICA": 283500000.01, "R-AMERICA": 36000000.0, "R-ASIA": 103500000.0, "R-EUROPE": 27000000.0} | 20170331 | 29979 | d000514001 | {"12262": 144000000.0, "12263": 81000000.0, "13040": 225000000.0} | 20150327 |
| 4652340566403237516 | 450000000 | 0 | {"R-AFRICA": 225000000.0, "R-AMERICA": 49500000.0, "R-ASIA": 126000000.0, "R-EUROPE": 49500000.0} | 20141231 | 29979 | m013354001 | {"12262": 99000000.0, "12263": 72000000.0, "13040": 279000000.0} | 20111209 |
| 286578574445610227 | 250000000.009999990463 | 0 | {"R-WESTINDIES": 50000000.0, "R-NORTHCENTRALAMERICA": 80000000.0, "R-SOUTHAMERICA": 120000000.0} | 20370331 | 29636 | m013705001 | {"14015": 25000000.0, "23210": 25000000.0, "23230": 25000000.0, "23240": 25000000.0, "23260": 250000 ... total length 189 | 20120330 |
| 6536815622391622287 | 930400000 | 0 | {"R-SOUTHSAHARA": 604760000.0, "R-SOUTHAMERICA": 37216000.0, "R-MIDDLEEAST": 74432000.0, "R-CENTRALA ... total length 140 | 20221231 | 29979 | p005247001 | {"12262": 279120000.0, "12263": 158168000.0, "13040": 493112000.0} | 20201216 |
| 6972508084776421254 | 250000000 | 0 | {"R-AFRICA": 50000000.0, "R-MIDDLEEAST": 75000000.0, "R-ASIA": 75000000.0, "R-EUROPE": 50000000.0} | 20470331 | 33915 | p010532001 | {"14021": 25000000.0, "23210": 175000000.0, "24010": 25000000.0, "41010": 25000000.0} | 20230330 |
| 7723023949484520621 | 450000000 | 0 | {"R-AFRICA": 283500000.0, "R-AMERICA": 31500000.0, "R-ASIA": 103500000.0, "R-EUROPE": 31500000.0} | 20110331 | 29979 | m012761001 | {"12262": 99000000.0, "12263": 72000000.0, "13040": 279000000.0} | 20080319 |

Total 19 rows
## 2_calc_quarterly_budget
### Script node 2_calc_quarterly_budget:
<pre id="json">{
  "2_calc_quarterly_budget": {
    "type": "table_custom_tfm_table",
    "custom_proc_type": "py_calc",
    "desc": "Calculate quarterly project budget for countries and sectors",
    "r": {
      "table": "project_financials",
      "expected_batches_total": 10
    },
    "p": {
      "python_code_urls": [
        "{dir_py}/calc_quarterly_budget.py"
      ],
      "calculated_fields": {
        "country_budget_json": {
          "expression": "map_to_quarterly_budget_json(r.start_date, r.end_date, r.country_amt_json)",
          "type": "string"
        },
        "sector_budget_json": {
          "expression": "map_to_quarterly_budget_json(r.start_date, r.end_date, r.sector_amt_json)",
          "type": "string"
        },
        "partner_budget_json": {
          "expression": "amt_to_quarterly_budget_json(r.start_date, r.end_date, r.amt)",
          "type": "string"
        }
      }
    },
    "w": {
      "name": "quarterly_project_budget",
      "fields": {
        "project_id": {
          "expression": "r.project_id",
          "type": "string"
        },
        "partner_id": {
          "expression": "r.partner_id",
          "type": "int"
        },
        "country_budget_json": {
          "expression": "p.country_budget_json",
          "type": "string"
        },
        "sector_budget_json": {
          "expression": "p.sector_budget_json",
          "type": "string"
        },
        "partner_budget_json": {
          "expression": "p.partner_budget_json",
          "type": "string"
        }
      }
    }
  }
}</pre>
### Script node 2_calc_quarterly_budget produces Cassandra table quarterly_project_budget:
| rowid | batch_idx | country_budget_json | partner_budget_json | partner_id | project_id | sector_budget_json |
| --- | --- | --- | --- | --- | --- | --- |
| 1859320587480934462 | 6 | {"R-AFRICA": {"2015-Q1": 1925951.0870244564, "2015-Q2": 35052309.783845104, "2015-Q3": 35437500.0012 ... total length 1040 | {"2015-Q1": 3057065.217459239, "2015-Q2": 55638586.95775815, "2015-Q3": 56250000.00125, "2015-Q4": 5 ... total length 257 | 29979 | d000514001 | {"12262": {"2015-Q1": 978260.8695652174, "2015-Q2": 17804347.826086957, "2015-Q3": 18000000.0, "2015 ... total length 766 |
| 3777753417576167724 | 7 | {"R-AFRICA": {"2020-Q1": 19661.582459485224, "2020-Q2": 596401.3346043852, "2020-Q3": 602955.1954242 ... total length 8472 | {"2020-Q1": 98307.91229742613, "2020-Q2": 2982006.6730219256, "2020-Q3": 3014775.9771210677, "2020-Q ... total length 2865 | 29578 | p006167001 | {"23210": {"2020-Q1": 80612.48808388942, "2020-Q2": 2445245.471877979, "2020-Q3": 2472116.3012392754 ... total length 5649 |
| 7247131808331670722 | 0 | {"R-AFRICA": {"2023-Q1": 11405.109489051094, "2023-Q2": 518932.4817518248, "2023-Q3": 524635.0364963 ... total length 11700 | {"2023-Q1": 57025.54744525548, "2023-Q2": 2594662.408759124, "2023-Q3": 2623175.182481752, "2023-Q4" ... total length 2928 | 33915 | p010532001 | {"14021": {"2023-Q1": 5702.554744525547, "2023-Q2": 259466.2408759124, "2023-Q3": 262317.51824817515 ... total length 11924 |
| 4423008682144623227 | 5 | {"R-AFRICA": {"2015-Q3": 2187119.234116623, "2015-Q4": 14372497.824194953, "2016-Q1": 14216275.02175 ... total length 2428 | {"2015-Q3": 3046127.0670147957, "2015-Q4": 20017406.440382943, "2016-Q1": 19799825.93559617, "2016-Q ... total length 790 | 29981 | d002243001 | {"12250": {"2015-Q3": 3046127.0670147957, "2015-Q4": 20017406.440382943, "2016-Q1": 19799825.9355961 ... total length 801 |
| 2110591658964151141 | 0 | {"R-SOUTHSAHARA": {"2020-Q4": 12970723.860589812, "2021-Q1": 72960321.71581769, "2021-Q2": 73770991. ... total length 1446 | {"2020-Q4": 19954959.78552279, "2021-Q1": 112246648.79356569, "2021-Q2": 113493833.78016086, "2021-Q ... total length 278 | 29979 | p005247001 | {"12262": {"2020-Q4": 5986487.935656836, "2021-Q1": 33673994.638069704, "2021-Q2": 34048150.13404825 ... total length 850 |
| 1081494116057048496 | 1 | {"R-SOUTHSAHARA": {"2019-Q3": 427830.59636992216, "2019-Q4": 1192739.8444252377, "2020-Q1": 1179775. ... total length 7948 | {"2019-Q3": 1711322.3854796886, "2019-Q4": 4770959.377700951, "2020-Q1": 4719101.123595505, "2020-Q2 ... total length 1909 | 30488 | p008157001 | {"15170": {"2019-Q3": 1711322.3854796886, "2019-Q4": 4770959.377700951, "2020-Q1": 4719101.123595505 ... total length 1920 |
| 96333662368837592 | 5 | {"R-SOUTHSAHARA": {"2023-Q4": 20348411.21495327, "2024-Q1": 102872523.36448598, "2024-Q2": 102872523 ... total length 1158 | {"2023-Q4": 29069158.878504675, "2024-Q1": 146960747.6635514, "2024-Q2": 146960747.6635514, "2024-Q3 ... total length 276 | 29979 | p012277001 | {"12262": {"2023-Q4": 8720747.663551401, "2024-Q1": 44088224.29906542, "2024-Q2": 44088224.29906542, ... total length 855 |
| 3434340729862472867 | 9 | {"R-WESTINDIES": {"2012-Q1": 10949.304719150334, "2012-Q2": 498193.3647213402, "2012-Q3": 503668.017 ... total length 9306 | {"2012-Q1": 54746.52359794153, "2012-Q2": 2490966.8237063396, "2012-Q3": 2518340.0855053104, "2012-Q ... total length 3130 | 29636 | m013705001 | {"14015": {"2012-Q1": 5474.652359575167, "2012-Q2": 249096.6823606701, "2012-Q3": 251834.00854045767 ... total length 27991 |
| 6692446017324546450 | 8 | {"R-AFRICA": {"2011-Q4": 4624664.879356569, "2012-Q1": 18297587.131367292, "2012-Q2": 18297587.13136 ... total length 1626 | {"2011-Q4": 9249329.758713137, "2012-Q1": 36595174.262734585, "2012-Q2": 36595174.262734585, "2012-Q ... total length 394 | 29979 | m013354001 | {"12262": {"2011-Q4": 2034852.54691689, "2012-Q1": 8050938.337801608, "2012-Q2": 8050938.337801608,  ... total length 1204 |
| 2662417924981044848 | 8 | {"R-AFRICA": {"2017-Q3": 13238240.0, "2017-Q4": 19643840.0, "2018-Q1": 19216800.0, "2018-Q2": 194303 ... total length 1007 | {"2017-Q3": 38936000.0, "2017-Q4": 57776000.0, "2018-Q1": 56520000.0, "2018-Q2": 57148000.0, "2018-Q ... total length 322 | 29979 | d003827001 | {"12262": {"2017-Q3": 11680800.0, "2017-Q4": 17332800.0, "2018-Q1": 16956000.0, "2018-Q2": 17144400. ... total length 985 |

Total 19 rows
## 3_tag_countries
### Script node 3_tag_countries:
<pre id="json">{
  "3_tag_countries": {
    "type": "table_custom_tfm_table",
    "custom_proc_type": "tag_and_denormalize",
    "desc": "Tag by country",
    "r": {
      "table": "quarterly_project_budget",
      "expected_batches_total": 10
    },
    "p": {
      "tag_field_name": "country_tag",
      "tag_criteria_url": "{dir_cfg}/tag_criteria_country.json"
    },
    "w": {
      "name": "quarterly_project_budget_tagged_by_country",
      "fields": {
        "project_id": {
          "expression": "r.project_id",
          "type": "string"
        },
        "country": {
          "expression": "p.country_tag",
          "type": "string"
        },
        "country_budget_json": {
          "expression": "r.country_budget_json",
          "type": "string"
        }
      }
    }
  }
}</pre>
### Script node 3_tag_countries produces Cassandra table quarterly_project_budget_tagged_by_country:
| rowid | batch_idx | country | country_budget_json | project_id |
| --- | --- | --- | --- | --- |
| 3315743091543635979 | 4 | R-AMERICA | {"R-AFRICA": {"2011-Q4": 4624664.879356569, "2012-Q1": 18297587.131367292, "2012-Q2": 18297587.13136 ... total length 1626 | m013354001 |
| 232657194511731982 | 4 | R-ASIA | {"R-SOUTHSAHARA": {"2023-Q4": 20348411.21495327, "2024-Q1": 102872523.36448598, "2024-Q2": 102872523 ... total length 1158 | p012277001 |
| 7305846314003206394 | 9 | R-AMERICA | {"R-AFRICA": {"2021-Q1": 301969.36542669585, "2021-Q2": 13739606.126914661, "2021-Q3": 13890590.8096 ... total length 1961 | p006992001 |
| 7034883004174972355 | 7 | DO | {"IN": {"2022-Q1": 22996.057818659658, "2022-Q2": 2092641.2614980289, "2022-Q3": 2115637.3193166885, ... total length 18632 | p010807001 |
| 7611719487153026897 | 3 | R-SOUTHSAHARA | {"R-SOUTHSAHARA": {"2020-Q4": 12970723.860589812, "2021-Q1": 72960321.71581769, "2021-Q2": 73770991. ... total length 1446 | p005247001 |
| 1036330799143506077 | 2 | R-AFRICA | {"R-AFRICA": {"2023-Q1": 11405.109489051094, "2023-Q2": 518932.4817518248, "2023-Q3": 524635.0364963 ... total length 11700 | p010532001 |
| 6391765181314034550 | 3 | R-AFRICA | {"R-SOUTHSAHARA": {"2019-Q3": 427830.59636992216, "2019-Q4": 1192739.8444252377, "2020-Q1": 1179775. ... total length 7948 | p008157001 |
| 7299612912787621919 | 5 | R-AMERICA | {"R-AFRICA": {"2015-Q1": 1925951.0870244564, "2015-Q2": 35052309.783845104, "2015-Q3": 35437500.0012 ... total length 1040 | d000514001 |
| 840984600135908957 | 2 | R-ASIA | {"R-AFRICA": {"2017-Q3": 13238240.0, "2017-Q4": 19643840.0, "2018-Q1": 19216800.0, "2018-Q2": 194303 ... total length 1007 | d003827001 |
| 1730201400797480307 | 7 | R-WESTINDIES | {"R-WESTINDIES": {"2024-Q1": 142450.14245014245, "2024-Q2": 540123.4567901234, "2024-Q3": 546058.879 ... total length 8602 | p012183001 |

Total 63 rows
## 3_tag_sectors
### Script node 3_tag_sectors:
<pre id="json">{
  "3_tag_sectors": {
    "type": "table_custom_tfm_table",
    "custom_proc_type": "tag_and_denormalize",
    "desc": "Tag by sector",
    "r": {
      "table": "quarterly_project_budget",
      "expected_batches_total": 10
    },
    "p": {
      "tag_field_name": "sector_tag",
      "tag_criteria_url": "{dir_cfg}/tag_criteria_sector.json"
    },
    "w": {
      "name": "quarterly_project_budget_tagged_by_sector",
      "fields": {
        "project_id": {
          "expression": "r.project_id",
          "type": "string"
        },
        "sector": {
          "expression": "int(p.sector_tag)",
          "type": "int"
        },
        "sector_budget_json": {
          "expression": "r.sector_budget_json",
          "type": "string"
        }
      }
    }
  }
}</pre>
### Script node 3_tag_sectors produces Cassandra table quarterly_project_budget_tagged_by_sector:
| rowid | batch_idx | project_id | sector | sector_budget_json |
| --- | --- | --- | --- | --- |
| 3820643885914075538 | 1 | p012801001 | 23630 | {"14021": {"2024-Q1": 81525.69515213277, "2024-Q2": 370941.9129422041, "2024-Q3": 375018.1976998107, ... total length 18663 |
| 1545774572741838340 | 4 | m013705001 | 23230 | {"14015": {"2012-Q1": 5474.652359575167, "2012-Q2": 249096.6823606701, "2012-Q3": 251834.00854045767 ... total length 27991 |
| 4669106684666863821 | 7 | m012672001 | 23630 | {"11420": {"2007-Q4": 1389933.2615715822, "2008-Q1": 2073506.9967707212, "2008-Q2": 2073506.99677072 ... total length 2429 |
| 8349889417835952966 | 7 | m012672001 | 21020 | {"11420": {"2007-Q4": 1389933.2615715822, "2008-Q1": 2073506.9967707212, "2008-Q2": 2073506.99677072 ... total length 2429 |
| 8187261670528356682 | 2 | p010532001 | 23210 | {"14021": {"2023-Q1": 5702.554744525547, "2023-Q2": 259466.2408759124, "2023-Q3": 262317.51824817515 ... total length 11924 |
| 3693664794125228841 | 1 | p006692001 | 11220 | {"11110": {"2022-Q1": 17063.74758531874, "2022-Q2": 1552801.0302640053, "2022-Q3": 1569864.777849324 ... total length 2225 |
| 8980965113080216996 | 4 | p012277001 | 12262 | {"12262": {"2023-Q4": 8720747.663551401, "2024-Q1": 44088224.29906542, "2024-Q2": 44088224.29906542, ... total length 855 |
| 7307591181337376132 | 4 | m013705001 | 14015 | {"14015": {"2012-Q1": 5474.652359575167, "2012-Q2": 249096.6823606701, "2012-Q3": 251834.00854045767 ... total length 27991 |
| 7218913746506800687 | 4 | m013354001 | 12263 | {"12262": {"2011-Q4": 2034852.54691689, "2012-Q1": 8050938.337801608, "2012-Q2": 8050938.337801608,  ... total length 1204 |
| 6633123094608569930 | 2 | d003827001 | 12263 | {"12262": {"2017-Q3": 11680800.0, "2017-Q4": 17332800.0, "2018-Q1": 16956000.0, "2018-Q2": 17144400. ... total length 985 |

Total 89 rows
## 4_tag_countries_quarter
### Script node 4_tag_countries_quarter:
<pre id="json">{
  "4_tag_countries_quarter": {
    "type": "table_custom_tfm_table",
    "custom_proc_type": "tag_and_denormalize",
    "desc": "Tag by country and quarter",
    "r": {
      "table": "quarterly_project_budget_tagged_by_country",
      "expected_batches_total": 10
    },
    "p": {
      "tag_field_name": "quarter_tag",
      "tag_criteria_url": "{dir_cfg}/tag_criteria_country_quarter.json"
    },
    "w": {
      "name": "quarterly_project_budget_tagged_by_country_quarter",
      "fields": {
        "project_id": {
          "expression": "r.project_id",
          "type": "string"
        },
        "country": {
          "expression": "r.country",
          "type": "string"
        },
        "quarter": {
          "expression": "p.quarter_tag",
          "type": "string"
        },
        "country_budget_json": {
          "expression": "r.country_budget_json",
          "type": "string"
        }
      }
    }
  }
}</pre>
### Script node 4_tag_countries_quarter produces Cassandra table quarterly_project_budget_tagged_by_country_quarter:
| rowid | batch_idx | country | country_budget_json | project_id | quarter |
| --- | --- | --- | --- | --- | --- |
| 4715004269345111126 | 6 | R-AFRICA | {"R-AFRICA": {"2020-Q1": 19661.582459485224, "2020-Q2": 596401.3346043852, "2020-Q3": 602955.1954242 ... total length 8472 | p006167001 | 2021-Q3 |
| 1021868600417521226 | 4 | PH | {"IN": {"2022-Q1": 22996.057818659658, "2022-Q2": 2092641.2614980289, "2022-Q3": 2115637.3193166885, ... total length 18632 | p010807001 | 2024-Q1 |
| 8016401234334020436 | 3 | R-ASIA | {"R-AFRICA": {"2017-Q3": 13238240.0, "2017-Q4": 19643840.0, "2018-Q1": 19216800.0, "2018-Q2": 194303 ... total length 1007 | d003827001 | 2019-Q4 |
| 8173111997092742705 | 0 | R-ASIA | {"R-ASIA": {"2024-Q1": 917164.0704614937, "2024-Q2": 4173096.5205997964, "2024-Q3": 4218954.72412287 ... total length 4700 | p012801001 | 2039-Q4 |
| 2584141339630182600 | 3 | R-ASIA | {"R-AFRICA": {"2023-Q1": 11405.109489051094, "2023-Q2": 518932.4817518248, "2023-Q3": 524635.0364963 ... total length 11700 | p010532001 | 2023-Q1 |
| 1257284422288393491 | 3 | R-AFRICA | {"R-SOUTHSAHARA": {"2019-Q3": 427830.59636992216, "2019-Q4": 1192739.8444252377, "2020-Q1": 1179775. ... total length 7948 | p008157001 | 2020-Q4 |
| 1559510570758845169 | 8 | R-WESTINDIES | {"R-WESTINDIES": {"2012-Q1": 10949.304719150334, "2012-Q2": 498193.3647213402, "2012-Q3": 503668.017 ... total length 9306 | m013705001 | 2013-Q2 |
| 2508231656225447287 | 6 | R-AFRICA | {"R-AFRICA": {"2020-Q1": 19661.582459485224, "2020-Q2": 596401.3346043852, "2020-Q3": 602955.1954242 ... total length 8472 | p006167001 | 2038-Q4 |
| 4385437186669948493 | 7 | DO | {"IN": {"2022-Q1": 22996.057818659658, "2022-Q2": 2092641.2614980289, "2022-Q3": 2115637.3193166885, ... total length 18632 | p010807001 | 2024-Q3 |
| 7830909385194525457 | 7 | R-AFRICA | {"R-AFRICA": {"2023-Q1": 11405.109489051094, "2023-Q2": 518932.4817518248, "2023-Q3": 524635.0364963 ... total length 11700 | p010532001 | 2039-Q4 |

Total 2815 rows
## 4_tag_sectors_quarter
### Script node 4_tag_sectors_quarter:
<pre id="json">{
  "4_tag_sectors_quarter": {
    "type": "table_custom_tfm_table",
    "custom_proc_type": "tag_and_denormalize",
    "desc": "Tag by sector and quarter",
    "r": {
      "table": "quarterly_project_budget_tagged_by_sector",
      "expected_batches_total": 10
    },
    "p": {
      "tag_field_name": "quarter_tag",
      "tag_criteria_url": "{dir_cfg}/tag_criteria_sector_quarter.json"
    },
    "w": {
      "name": "quarterly_project_budget_tagged_by_sector_quarter",
      "fields": {
        "project_id": {
          "expression": "r.project_id",
          "type": "string"
        },
        "sector": {
          "expression": "r.sector",
          "type": "int"
        },
        "quarter": {
          "expression": "p.quarter_tag",
          "type": "string"
        },
        "sector_budget_json": {
          "expression": "r.sector_budget_json",
          "type": "string"
        }
      }
    }
  }
}</pre>
### Script node 4_tag_sectors_quarter produces Cassandra table quarterly_project_budget_tagged_by_sector_quarter:
| rowid | batch_idx | project_id | quarter | sector | sector_budget_json |
| --- | --- | --- | --- | --- | --- |
| 6871610342243656549 | 0 | p012801001 | 2036-Q4 | 23630 | {"14021": {"2024-Q1": 81525.69515213277, "2024-Q2": 370941.9129422041, "2024-Q3": 375018.1976998107, ... total length 18663 |
| 3849398586719196129 | 3 | m013512001 | 2013-Q1 | 22020 | {"14021": {"2011-Q1": 749203.7684955753, "2011-Q2": 1311106.5948672567, "2011-Q3": 1325514.359646017 ... total length 6564 |
| 3941738223354499541 | 6 | p010807001 | 2039-Q4 | 23110 | {"23110": {"2022-Q1": 10950.503723171267, "2022-Q2": 996495.8388085853, "2022-Q3": 1007446.342531756 ... total length 6063 |
| 4901988133504573643 | 0 | p010532001 | 2044-Q2 | 24010 | {"14021": {"2023-Q1": 5702.554744525547, "2023-Q2": 259466.2408759124, "2023-Q3": 262317.51824817515 ... total length 11924 |
| 1567671269685032712 | 9 | m013705001 | 2015-Q3 | 23210 | {"14015": {"2012-Q1": 5474.652359575167, "2012-Q2": 249096.6823606701, "2012-Q3": 251834.00854045767 ... total length 27991 |
| 2652879578660303014 | 1 | m013705001 | 2013-Q3 | 14015 | {"14015": {"2012-Q1": 5474.652359575167, "2012-Q2": 249096.6823606701, "2012-Q3": 251834.00854045767 ... total length 27991 |
| 8950353874973931018 | 8 | p010807001 | 2039-Q4 | 23210 | {"23110": {"2022-Q1": 10950.503723171267, "2022-Q2": 996495.8388085853, "2022-Q3": 1007446.342531756 ... total length 6063 |
| 4986203116101121356 | 7 | p012801001 | 2027-Q3 | 24030 | {"14021": {"2024-Q1": 81525.69515213277, "2024-Q2": 370941.9129422041, "2024-Q3": 375018.1976998107, ... total length 18663 |
| 4152662510034510722 | 6 | p012183001 | 2038-Q1 | 23630 | {"21012": {"2024-Q1": 71225.07122507122, "2024-Q2": 270061.7283950617, "2024-Q3": 273029.43969610636 ... total length 31222 |
| 493379895319512279 | 3 | p012183001 | 2041-Q4 | 23270 | {"21012": {"2024-Q1": 71225.07122507122, "2024-Q2": 270061.7283950617, "2024-Q3": 273029.43969610636 ... total length 31222 |

Total 4100 rows
## 4_tag_partners_quarter
### Script node 4_tag_partners_quarter:
<pre id="json">{
  "4_tag_partners_quarter": {
    "type": "table_custom_tfm_table",
    "custom_proc_type": "tag_and_denormalize",
    "desc": "Tag by partner and quarter",
    "r": {
      "table": "quarterly_project_budget",
      "expected_batches_total": 10
    },
    "p": {
      "tag_field_name": "quarter_tag",
      "tag_criteria_url": "{dir_cfg}/tag_criteria_partner_quarter.json"
    },
    "w": {
      "name": "quarterly_project_budget_tagged_by_partner_quarter",
      "fields": {
        "project_id": {
          "expression": "r.project_id",
          "type": "string"
        },
        "partner_id": {
          "expression": "r.partner_id",
          "type": "int"
        },
        "quarter": {
          "expression": "p.quarter_tag",
          "type": "string"
        },
        "partner_budget_json": {
          "expression": "r.partner_budget_json",
          "type": "string"
        }
      }
    }
  }
}</pre>
### Script node 4_tag_partners_quarter produces Cassandra table quarterly_project_budget_tagged_by_partner_quarter:
| rowid | batch_idx | partner_budget_json | partner_id | project_id | quarter |
| --- | --- | --- | --- | --- | --- |
| 4627403874359917549 | 2 | {"2023-Q1": 57025.54744525548, "2023-Q2": 2594662.408759124, "2023-Q3": 2623175.182481752, "2023-Q4" ... total length 2928 | 33915 | p010532001 | 2028-Q1 |
| 7239814794359227421 | 7 | {"2024-Q1": 1424501.4245014247, "2024-Q2": 5401234.567901235, "2024-Q3": 5460588.793922127, "2024-Q4 ... total length 2791 | 29636 | p012183001 | 2040-Q2 |
| 4975517084762801605 | 5 | {"2020-Q1": 98307.91229742613, "2020-Q2": 2982006.6730219256, "2020-Q3": 3014775.9771210677, "2020-Q ... total length 2865 | 29578 | p006167001 | 2034-Q1 |
| 2115439484464325507 | 7 | {"2024-Q1": 1424501.4245014247, "2024-Q2": 5401234.567901235, "2024-Q3": 5460588.793922127, "2024-Q4 ... total length 2791 | 29636 | p012183001 | 2039-Q2 |
| 3137865540796549617 | 3 | {"2019-Q3": 1711322.3854796886, "2019-Q4": 4770959.377700951, "2020-Q1": 4719101.123595505, "2020-Q2 ... total length 1909 | 30488 | p008157001 | 2034-Q2 |
| 4946874832702262166 | 7 | {"2024-Q1": 1424501.4245014247, "2024-Q2": 5401234.567901235, "2024-Q3": 5460588.793922127, "2024-Q4 ... total length 2791 | 29636 | p012183001 | 2035-Q4 |
| 4483743663880167106 | 2 | {"2023-Q1": 57025.54744525548, "2023-Q2": 2594662.408759124, "2023-Q3": 2623175.182481752, "2023-Q4" ... total length 2928 | 33915 | p010532001 | 2035-Q1 |
| 2139984572353543545 | 7 | {"2022-Q1": 109505.03723171266, "2022-Q2": 9964958.388085851, "2022-Q3": 10074463.425317565, "2022-Q ... total length 3062 | 29578 | p010807001 | 2022-Q1 |
| 8402238031933271710 | 5 | {"2020-Q1": 98307.91229742613, "2020-Q2": 2982006.6730219256, "2020-Q3": 3014775.9771210677, "2020-Q ... total length 2865 | 29578 | p006167001 | 2041-Q2 |
| 2354898583089365024 | 1 | {"2022-Q1": 170637.47585318738, "2022-Q2": 15528010.302640053, "2022-Q3": 15698647.778493239, "2022- ... total length 558 | 30039 | p006692001 | 2023-Q1 |

Total 807 rows
## 5_project_country_quarter_amt
### Script node 5_project_country_quarter_amt:
<pre id="json">{
  "5_project_country_quarter_amt": {
    "type": "table_custom_tfm_table",
    "custom_proc_type": "py_calc",
    "desc": "Get country quarter amount",
    "r": {
      "table": "quarterly_project_budget_tagged_by_country_quarter",
      "expected_batches_total": "{get_amt_from_json_batches|number}"
    },
    "p": {
      "python_code_urls": [
        "{dir_py}/get_amt_from_json.py"
      ],
      "calculated_fields": {
        "amt": {
          "expression": "get_amt_by_key_and_quarter(r.country, r.quarter, r.country_budget_json)",
          "type": "float"
        }
      }
    },
    "w": {
      "name": "project_country_quarter_amt",
      "fields": {
        "project_id": {
          "expression": "r.project_id",
          "type": "string"
        },
        "country": {
          "expression": "r.country",
          "type": "string"
        },
        "quarter": {
          "expression": "r.quarter",
          "type": "string"
        },
        "amt": {
          "expression": "p.amt",
          "type": "float"
        }
      }
    }
  }
}</pre>
### Script node 5_project_country_quarter_amt produces Cassandra table project_country_quarter_amt:
| rowid | amt | batch_idx | country | project_id | quarter |
| --- | --- | --- | --- | --- | --- |
| 1178748811675745255 | 4589640.750670241192 | 6 | R-SOUTHAMERICA | p005247001 | 2022-Q4 |
| 3601558097913365059 | 2115637.319316688459 | 4 | ZA | p010807001 | 2027-Q3 |
| 1807772087307360 | 778398.72262773721 | 0 | R-ASIA | p010532001 | 2042-Q2 |
| 6510657118882702501 | 4218954.724122870713 | 4 | R-ASIA | p012801001 | 2029-Q3 |
| 3915251033927440768 | 1638176.638176638167 | 6 | R-NORTHCENTRALAMERICA | p012183001 | 2030-Q3 |
| 6404756265710794645 | 463677.39117775514 | 5 | R-OCEANIA | p012801001 | 2036-Q1 |
| 5834564234551792766 | 786952.554744525463 | 2 | R-ASIA | p010532001 | 2025-Q3 |
| 6669450010550111776 | 797196.671046868199 | 3 | DO | p010807001 | 2030-Q2 |
| 2702432518605190503 | 4425587.467362923548 | 6 | R-ASIA | d002243001 | 2019-Q1 |
| 5516819534943323641 | 3105602.060528010596 | 3 | R-ASIA | p006692001 | 2024-Q1 |

Total 2815 rows
## 5_project_sector_quarter_amt
### Script node 5_project_sector_quarter_amt:
<pre id="json">{
  "5_project_sector_quarter_amt": {
    "type": "table_custom_tfm_table",
    "custom_proc_type": "py_calc",
    "desc": "Get sector quarter amount",
    "r": {
      "table": "quarterly_project_budget_tagged_by_sector_quarter",
      "expected_batches_total": "{get_amt_from_json_batches|number}"
    },
    "p": {
      "python_code_urls": [
        "{dir_py}/get_amt_from_json.py"
      ],
      "calculated_fields": {
        "amt": {
          "expression": "get_amt_by_key_and_quarter(r.sector, r.quarter, r.sector_budget_json)",
          "type": "float"
        }
      }
    },
    "w": {
      "name": "project_sector_quarter_amt",
      "fields": {
        "project_id": {
          "expression": "r.project_id",
          "type": "string"
        },
        "sector": {
          "expression": "r.sector",
          "type": "int"
        },
        "quarter": {
          "expression": "r.quarter",
          "type": "string"
        },
        "amt": {
          "expression": "p.amt",
          "type": "float"
        }
      }
    }
  }
}</pre>
### Script node 5_project_sector_quarter_amt produces Cassandra table project_sector_quarter_amt:
| rowid | amt | batch_idx | project_id | quarter | sector |
| --- | --- | --- | --- | --- | --- |
| 5977762390222817657 | 251834.008540457668 | 3 | m013705001 | 2033-Q3 | 23260 |
| 8705007126095378056 | 251834.008540457668 | 1 | m013705001 | 2032-Q3 | 23270 |
| 1816698236218498254 | 249096.682360670093 | 6 | m013705001 | 2032-Q1 | 23240 |
| 4930853097135549914 | 562527.296549716149 | 8 | p012801001 | 2027-Q4 | 31150 |
| 9118134289610063356 | 382241.215574548871 | 7 | p012183001 | 2034-Q3 | 41040 |
| 6484634157243085497 | 246359.356180882518 | 2 | m013705001 | 2022-Q1 | 23270 |
| 2685543556272734147 | 2994703.982777179684 | 5 | m012672001 | 2009-Q3 | 14021 |
| 5566883227708984209 | 550298.44227689621 | 7 | p012801001 | 2029-Q1 | 31150 |
| 8032135523229813310 | 503668.017080915335 | 8 | m013705001 | 2021-Q3 | 41020 |
| 488108597161857763 | 4770959.37770095095 | 8 | p008157001 | 2031-Q3 | 15170 |

Total 4100 rows
## 5_project_partner_quarter_amt
### Script node 5_project_partner_quarter_amt:
<pre id="json">{
  "5_project_partner_quarter_amt": {
    "type": "table_custom_tfm_table",
    "custom_proc_type": "py_calc",
    "desc": "Get partner quarter amount",
    "r": {
      "table": "quarterly_project_budget_tagged_by_partner_quarter",
      "expected_batches_total": "{get_amt_from_json_batches|number}"
    },
    "p": {
      "python_code_urls": [
        "{dir_py}/get_amt_from_json.py"
      ],
      "calculated_fields": {
        "amt": {
          "expression": "get_amt_by_quarter(r.quarter, r.partner_budget_json)",
          "type": "float"
        }
      }
    },
    "w": {
      "name": "project_partner_quarter_amt",
      "fields": {
        "project_id": {
          "expression": "r.project_id",
          "type": "string"
        },
        "partner_id": {
          "expression": "r.partner_id",
          "type": "int"
        },
        "quarter": {
          "expression": "r.quarter",
          "type": "string"
        },
        "amt": {
          "expression": "p.amt",
          "type": "float"
        }
      }
    }
  }
}</pre>
### Script node 5_project_partner_quarter_amt produces Cassandra table project_country_quarter_amt:
| rowid | amt | batch_idx | country | project_id | quarter |
| --- | --- | --- | --- | --- | --- |
| 8751222469725115782 | 786952.554744525463 | 0 | R-ASIA | p010532001 | 2026-Q3 |
| 8063360272440380976 | 805957.074025405222 | 9 | DO | p010807001 | 2033-Q4 |
| 5195372877338837710 | 1491003.336510962807 | 6 | R-ASIA | p006167001 | 2036-Q1 |
| 373270937839131181 | 1179775.280898876255 | 8 | R-ASIA | p008157001 | 2024-Q2 |
| 1829940362690899251 | 540123.456790123368 | 1 | R-WESTINDIES | p012183001 | 2034-Q2 |
| 7432803228807627116 | 805957.074025405222 | 4 | DO | p010807001 | 2045-Q3 |
| 152607732623798964 | 904432.79313632031 | 4 | R-AMERICA | p006167001 | 2041-Q3 |
| 8126167813690408930 | 524635.036496350309 | 7 | R-AFRICA | p010532001 | 2044-Q4 |
| 4769882013062139717 | 805957.074025405222 | 8 | DO | p010807001 | 2046-Q4 |
| 4180165531138342676 | 1192739.844425237738 | 4 | R-ASIA | p008157001 | 2024-Q4 |

Total 2815 rows
## 6_file_project_country_quarter_amt
### Script node 6_file_project_country_quarter_amt:
<pre id="json">{
  "6_file_project_country_quarter_amt": {
    "type": "table_file",
    "desc": "Write ...",
    "r": {
      "table": "project_country_quarter_amt"
    },
    "w": {
      "top": {
        "order": "quarter,project_id,country"
      },
      "url_template": "{dir_out}/project_country_quarter_amt.csv",
      "columns": [
        {
          "csv": {
            "header": "project_id",
            "format": "%s"
          },
          "name": "project_id",
          "expression": "r.project_id",
          "type": "string"
        },
        {
          "csv": {
            "header": "country_code",
            "format": "%s"
          },
          "name": "country",
          "expression": "r.country",
          "type": "string"
        },
        {
          "csv": {
            "header": "quarter",
            "format": "%s"
          },
          "name": "quarter",
          "expression": "r.quarter",
          "type": "string"
        },
        {
          "csv": {
            "header": "amt",
            "format": "%.2f"
          },
          "name": "amt",
          "expression": "r.amt",
          "type": "float"
        }
      ]
    }
  }
}</pre>
### Script node 6_file_project_country_quarter_amt produces data file:
| project_id | country_code | quarter | amt |
| --- | --- | --- | --- |
| m012672001 | R-AFRICA | 2007-Q4 | 19856189.45 |
| m012672001 | R-AFRICA | 2008-Q1 | 29621528.53 |
| m012761001 | R-AFRICA | 2008-Q1 | 3326263.54 |
| m012761001 | R-AMERICA | 2008-Q1 | 369584.84 |
| m012761001 | R-ASIA | 2008-Q1 | 1214350.18 |
| m012761001 | R-EUROPE | 2008-Q1 | 369584.84 |
| m012672001 | R-AFRICA | 2008-Q2 | 29621528.53 |
| m012761001 | R-AFRICA | 2008-Q2 | 23283844.77 |
| m012761001 | R-AMERICA | 2008-Q2 | 2587093.86 |
| m012761001 | R-ASIA | 2008-Q2 | 8500451.26 |

Total 2815 rows
## 6_file_project_sector_quarter_amt
### Script node 6_file_project_sector_quarter_amt:
<pre id="json">{
  "6_file_project_sector_quarter_amt": {
    "type": "table_file",
    "desc": "Write ...",
    "r": {
      "table": "project_sector_quarter_amt"
    },
    "w": {
      "top": {
        "order": "quarter,project_id,sector"
      },
      "url_template": "{dir_out}/project_sector_quarter_amt.csv",
      "columns": [
        {
          "csv": {
            "header": "project_id",
            "format": "%s"
          },
          "name": "project_id",
          "expression": "r.project_id",
          "type": "string"
        },
        {
          "csv": {
            "header": "sector_id",
            "format": "%d"
          },
          "name": "sector",
          "expression": "r.sector",
          "type": "int"
        },
        {
          "csv": {
            "header": "quarter",
            "format": "%s"
          },
          "name": "quarter",
          "expression": "r.quarter",
          "type": "string"
        },
        {
          "csv": {
            "header": "amt",
            "format": "%.2f"
          },
          "name": "amt",
          "expression": "r.amt",
          "type": "float"
        }
      ]
    }
  }
}</pre>
### Script node 6_file_project_sector_quarter_amt produces data file:
| project_id | sector_id | quarter | amt |
| --- | --- | --- | --- |
| m012672001 | 11420 | 2007-Q4 | 1389933.26 |
| m012672001 | 14021 | 2007-Q4 | 1985618.95 |
| m012672001 | 14022 | 2007-Q4 | 1985618.95 |
| m012672001 | 15110 | 2007-Q4 | 3971237.89 |
| m012672001 | 21020 | 2007-Q4 | 3971237.89 |
| m012672001 | 23630 | 2007-Q4 | 3971237.89 |
| m012672001 | 31120 | 2007-Q4 | 2581304.63 |
| m012672001 | 11420 | 2008-Q1 | 2073507.00 |
| m012672001 | 14021 | 2008-Q1 | 2962152.85 |
| m012672001 | 14022 | 2008-Q1 | 2962152.85 |

Total 4100 rows
## 6_file_project_partner_quarter_amt
### Script node 6_file_project_partner_quarter_amt:
<pre id="json">{
  "6_file_project_partner_quarter_amt": {
    "type": "table_file",
    "desc": "Write ...",
    "r": {
      "table": "project_partner_quarter_amt"
    },
    "w": {
      "top": {
        "order": "quarter,project_id,partner_id"
      },
      "url_template": "{dir_out}/project_partner_quarter_amt.csv",
      "columns": [
        {
          "csv": {
            "header": "project_id",
            "format": "%s"
          },
          "name": "project_id",
          "expression": "r.project_id",
          "type": "string"
        },
        {
          "csv": {
            "header": "partner_id",
            "format": "%d"
          },
          "name": "partner_id",
          "expression": "r.partner_id",
          "type": "int"
        },
        {
          "csv": {
            "header": "quarter",
            "format": "%s"
          },
          "name": "quarter",
          "expression": "r.quarter",
          "type": "string"
        },
        {
          "csv": {
            "header": "amt",
            "format": "%.2f"
          },
          "name": "amt",
          "expression": "r.amt",
          "type": "float"
        }
      ]
    }
  }
}</pre>
### Script node 6_file_project_partner_quarter_amt produces data file:
| project_id | partner_id | quarter | amt |
| --- | --- | --- | --- |
| m012672001 | 29663 | 2007-Q4 | 19856189.45 |
| m012672001 | 29663 | 2008-Q1 | 29621528.53 |
| m012761001 | 29979 | 2008-Q1 | 5279783.39 |
| m012672001 | 29663 | 2008-Q2 | 29621528.53 |
| m012761001 | 29979 | 2008-Q2 | 36958483.75 |
| m012672001 | 29663 | 2008-Q3 | 29947039.83 |
| m012761001 | 29979 | 2008-Q3 | 37364620.94 |
| m012672001 | 29663 | 2008-Q4 | 29947039.83 |
| m012761001 | 29979 | 2008-Q4 | 37364620.94 |
| m012672001 | 29663 | 2009-Q1 | 29296017.22 |

Total 807 rows
