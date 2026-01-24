package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/capillariesio/capillaries/pkg/eval_capi"
	"github.com/capillariesio/capillaries/pkg/sc"
	"github.com/capillariesio/capillaries/pkg/storage"
	"github.com/shopspring/decimal"
	"golang.org/x/text/encoding/charmap"
)

func getColIdxMap(filePath string) (map[string]int, error) {
	m := map[string]int{
		"Reference Pool ID":                            -1,
		"Loan Identifier":                              -1,
		"Monthly Reporting Period":                     -1,
		"Channel":                                      -1,
		"Seller Name":                                  -1,
		"Servicer Name":                                -1,
		"Master Servicer":                              -1,
		"Original Interest Rate":                       -1,
		"Current Interest Rate":                        -1,
		"Original UPB":                                 -1,
		"UPB at Issuance":                              -1,
		"Current Actual UPB":                           -1,
		"Original Loan Term":                           -1,
		"Origination Date":                             -1,
		"First Payment Date":                           -1,
		"Loan Age":                                     -1,
		"Remaining Months to Legal Maturity":           -1,
		"Remaining Months To Maturity":                 -1,
		"Maturity Date":                                -1,
		"Original Loan to Value Ratio (LTV)":           -1,
		"Original Combined Loan to Value Ratio (CLTV)": -1,
		"Number of Borrowers":                          -1,
		"Debt-To-Income (DTI)":                         -1,
		"Borrower Credit Score at Origination":         -1,
		"Co-Borrower Credit Score at Origination":      -1,
		"First Time Home Buyer Indicator":              -1,
		"Loan Purpose ":                                -1,
		"Property Type":                                -1,
		"Number of Units":                              -1,
		"Occupancy Status":                             -1,
		"Property State":                               -1,
		"Metropolitan Statistical Area (MSA)":          -1,
		"Zip Code Short":                               -1,
		"Mortgage Insurance Percentage":                -1,
		"Amortization Type":                            -1,
		"Prepayment Penalty Indicator":                 -1,
		"Interest Only Loan Indicator":                 -1,
		"Interest Only First Principal And Interest Payment Date": -1,
		"Months to Amortization":                                  -1,
		"Current Loan Delinquency Status":                         -1,
		"Loan Payment History":                                    -1,
		"Modification Flag":                                       -1,
		"Mortgage Insurance Cancellation Indicator":               -1,
		"Zero Balance Code":                                       -1,
		"Zero Balance Effective Date":                             -1,
		"UPB at the Time of Removal":                              -1,
		"Repurchase Date":                                         -1,
		"Scheduled Principal Current":                             -1,
		"Total Principal Current":                                 -1,
		"Unscheduled Principal Current":                           -1,
		"Last Paid Installment Date":                              -1,
		"Foreclosure Date":                                        -1,
		"Disposition Date":                                        -1,
		"Foreclosure Costs":                                       -1,
		"Property Preservation and Repair Costs":                  -1,
		"Asset Recovery Costs":                                    -1,
		"Miscellaneous Holding Expenses and Credits":              -1,
		"Associated Taxes for Holding Property":                   -1,
		"Net Sales Proceeds":                                      -1,
		"Credit Enhancement Proceeds":                             -1,
		"Repurchase Make Whole Proceeds":                          -1,
		"Other Foreclosure Proceeds":                              -1,
		"Modification-Related Non-Interest Bearing UPB":           -1,
		"Principal Forgiveness Amount":                            -1,
		"Original List Start Date":                                -1,
		"Original List Price":                                     -1,
		"Current List Start Date":                                 -1,
		"Current List Price":                                      -1,
		"Borrower Credit Score At Issuance":                       -1,
		"Co-Borrower Credit Score At Issuance":                    -1,
		"Borrower Credit Score Current ":                          -1,
		"Co-Borrower Credit Score Current":                        -1,
		"Mortgage Insurance Type":                                 -1,
		"Servicing Activity Indicator":                            -1,
		"Current Period Modification Loss Amount":                 -1,
		"Cumulative Modification Loss Amount":                     -1,
		"Current Period Credit Event Net Gain or Loss":            -1,
		"Cumulative Credit Event Net Gain or Loss":                -1,
		"HomeReady® Program Indicator":                            -1,
		"Foreclosure Principal Write-off Amount":                  -1,
		"Relocation Mortgage Indicator":                           -1,
		"Zero Balance Code Change Date":                           -1,
		"Loan Holdback Indicator":                                 -1,
		"Loan Holdback Effective Date":                            -1,
		"Delinquent Accrued Interest":                             -1,
		"Property Valuation Method ":                              -1,
		"High Balance Loan Indicator ":                            -1,
		"ARM Initial Fixed-Rate Period  ? 5 YR Indicator":         -1,
		"ARM Product Type":                                        -1,
		"Initial Fixed-Rate Period ":                              -1,
		"Interest Rate Adjustment Frequency":                      -1,
		"Next Interest Rate Adjustment Date":                      -1,
		"Next Payment Change Date":                                -1,
		"Index":                                                   -1,
		"ARM Cap Structure":                                       -1,
		"Initial Interest Rate Cap Up Percent":                    -1,
		"Periodic Interest Rate Cap Up Percent":                   -1,
		"Lifetime Interest Rate Cap Up Percent":                   -1,
		"Mortgage Margin":                                         -1,
		"ARM Balloon Indicator":                                   -1,
		"ARM Plan Number":                                         -1,
		"Borrower Assistance Plan":                                -1,
		"High Loan to Value (HLTV) Refinance Option Indicator":    -1,
		"Deal Name":                                     -1,
		"Repurchase Make Whole Proceeds Flag":           -1,
		"Alternative Delinquency Resolution":            -1,
		"Alternative Delinquency  Resolution Count":     -1,
		"Total Deferral Amount":                         -1,
		"Payment Deferral Modification Event Indicator": -1,
		"Interest Bearing UPB":                          -1}

	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("cannot open file %s: %s", filePath, err.Error())
	}
	if f == nil {
		return nil, fmt.Errorf("cannot open file %s: unknown error", filePath)
	}
	defer f.Close()

	//fileReader := bufio.NewReader(f)
	//r := csv.NewReader(fileReader)
	r := csv.NewReader(charmap.ISO8859_15.NewDecoder().Reader(f))
	r.Comma = rune(',')
	// To avoid bare \" error: https://stackoverflow.com/questions/31326659/golang-csv-error-bare-in-non-quoted-field
	r.LazyQuotes = true

	line, err := r.Read()
	if err == io.EOF {
		return nil, fmt.Errorf("file %s does not have field names", filePath)
	}
	if err != nil {
		return nil, fmt.Errorf("cannot read field names from file %s: %s", filePath, err.Error())
	}

	for colIdx, colName := range line {
		if _, ok := m[colName]; !ok {
			return nil, fmt.Errorf("unknown column name %s", colName)
		}
		m[colName] = colIdx
	}

	for colName, colIdx := range m {
		if colIdx == -1 {
			return nil, fmt.Errorf("cannot find column index for column %s", colName)
		}
	}

	return m, nil
}

func printFileStatus(newElCounter int, newElCounterIncludingIrrelevant int, curFileName string) {
	if newElCounterIncludingIrrelevant > 0 {
		fmt.Printf("%d\t/\t%d\t%.0f%%\t%s\n", newElCounter, newElCounterIncludingIrrelevant, float64(newElCounter)*100/float64(newElCounterIncludingIrrelevant), curFileName)
	} else {
		fmt.Printf("0\t/\t0\t100%%\t%s\n", curFileName)
	}
}
func fannieMaeCsvToParquet(dealName string, files []string, colIdxMap map[string]int, outDir string) error {
	colTypeMap := map[string]eval_capi.TableFieldType{
		"Reference Pool ID":                            eval_capi.FieldTypeInt,
		"Loan Identifier":                              eval_capi.FieldTypeInt,
		"Monthly Reporting Period":                     eval_capi.FieldTypeInt,
		"Channel":                                      eval_capi.FieldTypeString,
		"Seller Name":                                  eval_capi.FieldTypeString,
		"Servicer Name":                                eval_capi.FieldTypeString,
		"Master Servicer":                              eval_capi.FieldTypeString,
		"Original Interest Rate":                       eval_capi.FieldTypeFloat,
		"Current Interest Rate":                        eval_capi.FieldTypeFloat,
		"Original UPB":                                 eval_capi.FieldTypeDecimal2,
		"UPB at Issuance":                              eval_capi.FieldTypeDecimal2,
		"Current Actual UPB":                           eval_capi.FieldTypeDecimal2,
		"Original Loan Term":                           eval_capi.FieldTypeInt,
		"Origination Date":                             eval_capi.FieldTypeInt,
		"First Payment Date":                           eval_capi.FieldTypeInt,
		"Loan Age":                                     eval_capi.FieldTypeInt,
		"Remaining Months to Legal Maturity":           eval_capi.FieldTypeInt,
		"Remaining Months To Maturity":                 eval_capi.FieldTypeInt,
		"Maturity Date":                                eval_capi.FieldTypeInt,
		"Original Loan to Value Ratio (LTV)":           eval_capi.FieldTypeInt,
		"Original Combined Loan to Value Ratio (CLTV)": eval_capi.FieldTypeInt,
		"Number of Borrowers":                          eval_capi.FieldTypeInt,
		"Debt-To-Income (DTI)":                         eval_capi.FieldTypeInt,
		"Borrower Credit Score at Origination":         eval_capi.FieldTypeInt,
		"Co-Borrower Credit Score at Origination":      eval_capi.FieldTypeInt,
		"First Time Home Buyer Indicator":              eval_capi.FieldTypeString,
		"Loan Purpose ":                                eval_capi.FieldTypeString,
		"Property Type":                                eval_capi.FieldTypeString,
		"Number of Units":                              eval_capi.FieldTypeInt,
		"Occupancy Status":                             eval_capi.FieldTypeString,
		"Property State":                               eval_capi.FieldTypeString,
		"Metropolitan Statistical Area (MSA)":          eval_capi.FieldTypeString,
		"Zip Code Short":                               eval_capi.FieldTypeString,
		"Mortgage Insurance Percentage":                eval_capi.FieldTypeFloat,
		"Amortization Type":                            eval_capi.FieldTypeString,
		"Prepayment Penalty Indicator":                 eval_capi.FieldTypeString,
		"Interest Only Loan Indicator":                 eval_capi.FieldTypeString,
		"Interest Only First Principal And Interest Payment Date": eval_capi.FieldTypeInt,
		"Months to Amortization":                                  eval_capi.FieldTypeInt,
		"Current Loan Delinquency Status":                         eval_capi.FieldTypeString,
		"Loan Payment History":                                    eval_capi.FieldTypeString,
		"Modification Flag":                                       eval_capi.FieldTypeString,
		"Mortgage Insurance Cancellation Indicator":               eval_capi.FieldTypeString,
		"Zero Balance Code":                                       eval_capi.FieldTypeString,
		"Zero Balance Effective Date":                             eval_capi.FieldTypeInt,
		"UPB at the Time of Removal":                              eval_capi.FieldTypeDecimal2,
		"Repurchase Date":                                         eval_capi.FieldTypeInt,
		"Scheduled Principal Current":                             eval_capi.FieldTypeDecimal2,
		"Total Principal Current":                                 eval_capi.FieldTypeDecimal2,
		"Unscheduled Principal Current":                           eval_capi.FieldTypeDecimal2,
		"Last Paid Installment Date":                              eval_capi.FieldTypeInt,
		"Foreclosure Date":                                        eval_capi.FieldTypeInt,
		"Disposition Date":                                        eval_capi.FieldTypeInt,
		"Foreclosure Costs":                                       eval_capi.FieldTypeInt,
		"Property Preservation and Repair Costs":                  eval_capi.FieldTypeDecimal2,
		"Asset Recovery Costs":                                    eval_capi.FieldTypeDecimal2,
		"Miscellaneous Holding Expenses and Credits":              eval_capi.FieldTypeDecimal2,
		"Associated Taxes for Holding Property":                   eval_capi.FieldTypeDecimal2,
		"Net Sales Proceeds":                                      eval_capi.FieldTypeDecimal2,
		"Credit Enhancement Proceeds":                             eval_capi.FieldTypeDecimal2,
		"Repurchase Make Whole Proceeds":                          eval_capi.FieldTypeDecimal2,
		"Other Foreclosure Proceeds":                              eval_capi.FieldTypeDecimal2,
		"Modification-Related Non-Interest Bearing UPB":           eval_capi.FieldTypeDecimal2,
		"Principal Forgiveness Amount":                            eval_capi.FieldTypeDecimal2,
		"Original List Start Date":                                eval_capi.FieldTypeInt,
		"Original List Price":                                     eval_capi.FieldTypeInt,
		"Current List Start Date":                                 eval_capi.FieldTypeDecimal2,
		"Current List Price":                                      eval_capi.FieldTypeDecimal2,
		"Borrower Credit Score At Issuance":                       eval_capi.FieldTypeInt,
		"Co-Borrower Credit Score At Issuance":                    eval_capi.FieldTypeInt,
		"Borrower Credit Score Current ":                          eval_capi.FieldTypeInt,
		"Co-Borrower Credit Score Current":                        eval_capi.FieldTypeInt,
		"Mortgage Insurance Type":                                 eval_capi.FieldTypeString,
		"Servicing Activity Indicator":                            eval_capi.FieldTypeString,
		"Current Period Modification Loss Amount":                 eval_capi.FieldTypeDecimal2,
		"Cumulative Modification Loss Amount":                     eval_capi.FieldTypeDecimal2,
		"Current Period Credit Event Net Gain or Loss":            eval_capi.FieldTypeDecimal2,
		"Cumulative Credit Event Net Gain or Loss":                eval_capi.FieldTypeDecimal2,
		"HomeReady® Program Indicator":                            eval_capi.FieldTypeString,
		"Foreclosure Principal Write-off Amount":                  eval_capi.FieldTypeDecimal2,
		"Relocation Mortgage Indicator":                           eval_capi.FieldTypeString,
		"Zero Balance Code Change Date":                           eval_capi.FieldTypeInt,
		"Loan Holdback Indicator":                                 eval_capi.FieldTypeString,
		"Loan Holdback Effective Date":                            eval_capi.FieldTypeInt,
		"Delinquent Accrued Interest":                             eval_capi.FieldTypeDecimal2,
		"Property Valuation Method ":                              eval_capi.FieldTypeString,
		"High Balance Loan Indicator ":                            eval_capi.FieldTypeString,
		"ARM Initial Fixed-Rate Period  ? 5 YR Indicator":         eval_capi.FieldTypeString,
		"ARM Product Type":                                        eval_capi.FieldTypeString,
		"Initial Fixed-Rate Period ":                              eval_capi.FieldTypeString,
		"Interest Rate Adjustment Frequency":                      eval_capi.FieldTypeString,
		"Next Interest Rate Adjustment Date":                      eval_capi.FieldTypeInt,
		"Next Payment Change Date":                                eval_capi.FieldTypeInt,
		"Index":                                                   eval_capi.FieldTypeString,
		"ARM Cap Structure":                                       eval_capi.FieldTypeString,
		"Initial Interest Rate Cap Up Percent":                    eval_capi.FieldTypeFloat,
		"Periodic Interest Rate Cap Up Percent":                   eval_capi.FieldTypeFloat,
		"Lifetime Interest Rate Cap Up Percent":                   eval_capi.FieldTypeFloat,
		"Mortgage Margin":                                         eval_capi.FieldTypeString,
		"ARM Balloon Indicator":                                   eval_capi.FieldTypeString,
		"ARM Plan Number":                                         eval_capi.FieldTypeString,
		"Borrower Assistance Plan":                                eval_capi.FieldTypeString,
		"High Loan to Value (HLTV) Refinance Option Indicator":    eval_capi.FieldTypeString,
		"Deal Name":                                     eval_capi.FieldTypeString,
		"Repurchase Make Whole Proceeds Flag":           eval_capi.FieldTypeString,
		"Alternative Delinquency Resolution":            eval_capi.FieldTypeString,
		"Alternative Delinquency  Resolution Count":     eval_capi.FieldTypeInt,
		"Total Deferral Amount":                         eval_capi.FieldTypeDecimal2,
		"Payment Deferral Modification Event Indicator": eval_capi.FieldTypeString,
		"Interest Bearing UPB":                          eval_capi.FieldTypeDecimal2}

	dateConvertSet := map[string]struct{}{
		"Monthly Reporting Period": struct{}{},
		"Maturity Date":            struct{}{},
		"Origination Date":         struct{}{},
		"First Payment Date":       struct{}{},
		"Interest Only First Principal And Interest Payment Date": struct{}{},
		"Zero Balance Effective Date":                             struct{}{},
		"Repurchase Date":                                         struct{}{},
		"Last Paid Installment Date":                              struct{}{},
		"Foreclosure Date":                                        struct{}{},
		"Disposition Date":                                        struct{}{},
		"Original List Start Date":                                struct{}{},
		"Current List Start Date":                                 struct{}{},
		"Zero Balance Code Change Date":                           struct{}{},
		"Loan Holdback Effective Date":                            struct{}{},
		"Next Interest Rate Adjustment Date":                      struct{}{},
		"Next Payment Change Date":                                struct{}{}}

	colSupportedMap := map[string]struct{}{
		"Loan Identifier":                    struct{}{},
		"Deal Name":                          struct{}{},
		"Origination Date":                   struct{}{},
		"Original UPB":                       struct{}{},
		"UPB at Issuance":                    struct{}{},
		"Original Loan Term":                 struct{}{},
		"Monthly Reporting Period":           struct{}{},
		"Current Actual UPB":                 struct{}{},
		"Remaining Months to Legal Maturity": struct{}{},
		"Remaining Months To Maturity":       struct{}{},
		"Zero Balance Effective Date":        struct{}{},
		"Scheduled Principal Current":        struct{}{},
		"Original Interest Rate":             struct{}{},
		"Seller Name":                        struct{}{},
		// "Servicer Name":                        struct{}{}, do not rely on it, it becomes empty the moment the mtge is paid off, and it can potentially change along the way (?)
		"Borrower Credit Score at Origination": struct{}{},
	}

	var fParquet *os.File
	var curFileName string
	fileCounter := 0
	var newElCounter int
	var newElCounterIncludingIrrelevant int
	var w *storage.ParquetWriter

	for _, srcFilePath := range files {
		f, err := os.Open(srcFilePath)
		if err != nil {
			return fmt.Errorf("cannot open file %s: %s", srcFilePath, err.Error())
		}

		fileReader := bufio.NewReader(f)
		r := csv.NewReader(fileReader)
		r.Comma = rune('|')
		// To avoid bare \" error: https://stackoverflow.com/questions/31326659/golang-csv-error-bare-in-non-quoted-field
		r.LazyQuotes = true

		for {
			line, err := r.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				return fmt.Errorf("cannot read from file %s: %s", srcFilePath, err.Error())
			}

			if fParquet == nil {
				curFileName = fmt.Sprintf("%s_%03d.parquet", strings.ReplaceAll(dealName, " ", "_"), fileCounter)
				fParquet, err = os.Create(path.Join(outDir, curFileName))
				if err != nil {
					return fmt.Errorf("cannot create file '%s': %s", curFileName, err.Error())
				}

				fileCounter++
				newElCounter = 0
				newElCounterIncludingIrrelevant = 0

				w, err = storage.NewParquetWriter(fParquet, sc.ParquetCodecGzip)
				if err != nil {
					return err
				}

				for colName, colType := range colTypeMap {
					if _, isSupported := colSupportedMap[colName]; isSupported {
						if err := w.AddColumn(colName, colType); err != nil {
							return err
						}
					}
				}
			}

			valMap := map[string]any{}

			for colName, colType := range colTypeMap {
				colIdx := colIdxMap[colName]
				if colIdx >= len(line) {
					// Whatever, a column is missing, it happens here
					continue
				}
				if _, isSupported := colSupportedMap[colName]; !isSupported {
					// We do not use this column, so do not write it to parquet
					continue
				}
				strVal := line[colIdx]
				switch colType {
				case eval_capi.FieldTypeInt:
					v := sc.DefaultInt
					if len(strVal) != 0 {
						v, err = strconv.ParseInt(strVal, 10, 64)
						if err != nil {
							return fmt.Errorf("cannot read int64 column %s from string '%s': %s", colName, strVal, err.Error())
						}
					}

					// Fannie Mae's MMDDYY -> Human readable YYYYMMDD
					if v != 0 {
						if _, isDate := dateConvertSet[colName]; isDate {
							y := (v % 100) + 2000
							m := v / 10000
							d := (v / 100) % 100
							v = y*10000 + m*100 + d
						}
					}
					valMap[colName] = v

				case eval_capi.FieldTypeString:
					// Check deal name
					if colName == "Deal Name" {
						if strVal == "" {
							strVal = dealName
						} else if strVal != dealName {
							return fmt.Errorf("unmatching deal name %s, expected %s", strVal, dealName)
						}
					}

					valMap[colName] = strVal

				case eval_capi.FieldTypeFloat:
					v := sc.DefaultFloat
					if len(strVal) != 0 {
						v, err = strconv.ParseFloat(strVal, 64)
						if err != nil {
							return fmt.Errorf("cannot read float64 column %s from string '%s': %s", colName, strVal, err.Error())
						}
					}
					valMap[colName] = v

				case eval_capi.FieldTypeDecimal2:
					v := sc.DefaultDecimal2()
					if len(strVal) != 0 {
						f, err := strconv.ParseFloat(strVal, 64)
						if err != nil {
							return fmt.Errorf("cannot read decimal column %s from string '%s': %s", colName, strVal, err.Error())
						}
						v = decimal.NewFromFloat(f)
					}
					valMap[colName] = storage.ParquetWriterDecimal2(v)

				default:
					return fmt.Errorf("cannot read column %s of unsupported type", colName)
				}
			}

			// Filter out payments beyond paid off date
			zeroBalEffDateVolatile, ok := valMap["Zero Balance Effective Date"]
			if !ok {
				return fmt.Errorf("cannot find Zero Balance Effective Date")
			}
			zeroBalEffDate, ok := zeroBalEffDateVolatile.(int64)
			if !ok {
				return fmt.Errorf("cannot convert Zero Balance Effective Date: %v,%t", zeroBalEffDateVolatile, zeroBalEffDateVolatile)
			}

			schedPrincipalCurrVolatile, ok := valMap["Scheduled Principal Current"]
			if !ok {
				return fmt.Errorf("cannot find Scheduled Principal Current")
			}
			// It's written to parquet as int64
			schedPrincipalCurr, ok := schedPrincipalCurrVolatile.(int64)
			if !ok {
				return fmt.Errorf("cannot convert Scheduled Principal Current: %v,%t", schedPrincipalCurrVolatile, schedPrincipalCurrVolatile)
			}

			newElCounterIncludingIrrelevant++

			if zeroBalEffDate > 0 && schedPrincipalCurr == int64(0) {
				continue
			}

			// Ids for quicktest data generation:
			// 2023_R01_G1 134238766 134240147
			// 2023_R02_G1 134597426 134597477
			// loanIdAny, _ := valMap["Loan Identifier"]
			// loanId, ok := loanIdAny.(int64)
			// if !ok {
			// 	return fmt.Errorf("aaa")
			// }
			// if loanId != 134238766 && loanId != 134240147 && loanId != 134597426 && loanId != 134597477 {
			// 	continue
			// }

			newElCounter++

			if err := w.FileWriter.AddData(valMap); err != nil {
				return fmt.Errorf("cannot write %v: %s", valMap, err.Error())
			}

			if newElCounter == 100000 {
				if err := w.Close(); err != nil {
					return fmt.Errorf("cannot close parquet writer '%s': %s", curFileName, err.Error())
				}

				if err := fParquet.Close(); err != nil {
					return fmt.Errorf("cannot close file '%s': %s", curFileName, err.Error())
				}

				printFileStatus(newElCounter, newElCounterIncludingIrrelevant, curFileName)

				fParquet = nil
				w = nil
			}
		}

	}

	if fParquet != nil {
		if err := w.Close(); err != nil {
			return fmt.Errorf("cannot close parquet writer '%s': %s", curFileName, err.Error())
		}

		if err := fParquet.Close(); err != nil {
			return fmt.Errorf("cannot close file '%s': %s", curFileName, err.Error())
		}

		printFileStatus(newElCounter, newElCounterIncludingIrrelevant, curFileName)
	}

	return nil
}

func main() {
	if len(os.Args) < 4 {
		fmt.Fprintln(os.Stderr, "required parameters: <deal name> <directory with CAS files> <output directory>")
		os.Exit(1)
	}

	colIdxMap, err := getColIdxMap(path.Join(os.Args[2], "CRT_Header_File.csv"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	files := []string{}
	err = filepath.Walk(os.Args[2], func(path string, f os.FileInfo, err error) error {
		if filepath.Ext(path) == ".csv" && strings.HasPrefix(filepath.Base(path), "CAS_") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	err = fannieMaeCsvToParquet(os.Args[1], files, colIdxMap, os.Args[3])
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	os.Exit(0)
}
