package main

import (
	"bufio"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/capillariesio/capillaries/pkg/sc"
	"github.com/capillariesio/capillaries/pkg/storage"
)

func readQuickAccounts(fileQuickAccountsPath string) (map[string]string, []string, error) {
	f, err := os.Open(fileQuickAccountsPath)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot open %s: %s", fileQuickAccountsPath, err.Error())
	}

	m := map[string]string{} // ARKK-> 2020-12-31
	r := csv.NewReader(bufio.NewReader(f))
	isHeader := true
	for {
		line, err := r.Read()
		if err == io.EOF {
			break
		}
		if isHeader {
			isHeader = false
			continue
		}
		m[line[0]] = line[1]
	}
	a := make([]string, len(m))
	i := 0
	for k, _ := range m {
		a[i] = k
		i++
	}
	return m, a, nil
}

func generateHoldings(fileQuickHoldingsPath string, fileInHoldingsPath string, bigAccountsMap map[string][]string, splitCount int) error {
	f, err := os.Open(fileQuickHoldingsPath)
	if err != nil {
		return fmt.Errorf("cannot open %s: %s", fileQuickHoldingsPath, err.Error())
	}

	fileCounter := 0

	var fParquet *os.File
	var w *storage.ParquetWriter
	var newElCounter int
	var curFilePath string

	r := csv.NewReader(bufio.NewReader(f))
	isHeader := true
	for {
		line, err := r.Read()
		if err == io.EOF {
			break
		}
		if isHeader {
			isHeader = false
			continue
		}

		accPrefix := line[0]
		accIds, ok := bigAccountsMap[accPrefix]
		if !ok {
			return fmt.Errorf("unknown account prefix '%s' in holdings", accPrefix)
		}
		for _, accId := range accIds {
			d, err := time.Parse("2006-01-02", line[1])
			if err != nil {
				return fmt.Errorf("cannot parse datetime '%s' in holdings: %s", line[1], err.Error())
			}
			qty, err := strconv.ParseInt(line[3], 10, 32)
			if err != nil {
				return fmt.Errorf("cannot parse qty '%s' in holdings: %s", line[3], err.Error())
			}

			if fParquet == nil {
				curFilePath = fmt.Sprintf("%s_%03d.parquet", fileInHoldingsPath, fileCounter)
				fParquet, err = os.Create(curFilePath)
				if err != nil {
					return fmt.Errorf("cannot create file '%s': %s", curFilePath, err.Error())
				}

				fileCounter++
				newElCounter = 0

				w, err = storage.NewParquetWriter(fParquet, sc.ParquetCodecGzip)
				if err != nil {
					return err
				}

				if err := w.AddColumn("account_id", sc.FieldTypeString); err != nil {
					return err
				}
				if err := w.AddColumn("d", sc.FieldTypeDateTime); err != nil {
					return err
				}
				if err := w.AddColumn("ticker", sc.FieldTypeString); err != nil {
					return err
				}
				if err := w.AddColumn("qty", sc.FieldTypeInt); err != nil {
					return err
				}
			}

			if err := w.FileWriter.AddData(map[string]any{
				"account_id": accId,
				"d":          storage.ParquetWriterMilliTs(d),
				"ticker":     line[2],
				"qty":        qty,
			}); err != nil {
				return fmt.Errorf("cannot write '%s' to holdings: %s", accId, err.Error())
			}

			newElCounter++

			if newElCounter == splitCount {
				if err := w.Close(); err != nil {
					return fmt.Errorf("cannot close parquet writer '%s': %s", curFilePath, err.Error())
				}

				if err := fParquet.Close(); err != nil {
					return fmt.Errorf("cannot close file '%s': %s", curFilePath, err.Error())
				}

				fParquet = nil
				w = nil
			}
		}
	}

	if fParquet != nil {
		if err := w.Close(); err != nil {
			return fmt.Errorf("cannot close parquet writer '%s': %s", curFilePath, err.Error())
		}

		if err := fParquet.Close(); err != nil {
			return fmt.Errorf("cannot close file '%s': %s", curFilePath, err.Error())
		}
	}

	return nil
}

func generateTxns(fileQuickTxnsPath string, fileInTxnsPath string, bigAccountsMap map[string][]string, splitCount int) error {
	f, err := os.Open(fileQuickTxnsPath)
	if err != nil {
		return fmt.Errorf("cannot open %s: %s", fileQuickTxnsPath, err.Error())
	}

	fileCounter := 0

	var fParquet *os.File
	var w *storage.ParquetWriter
	var newElCounter int
	var curFilePath string

	r := csv.NewReader(bufio.NewReader(f))
	isHeader := true
	for {
		line, err := r.Read()
		if err == io.EOF {
			break
		}
		if isHeader {
			isHeader = false
			continue
		}

		accPrefix := line[1]
		accIds, ok := bigAccountsMap[accPrefix]
		if !ok {
			return fmt.Errorf("unknown account prefix '%s' in txns", accPrefix)
		}
		for _, accId := range accIds {
			d, err := time.Parse("2006-01-02", line[0])
			if err != nil {
				return fmt.Errorf("cannot parse datetime '%s' in txns: %s", line[1], err.Error())
			}
			qty, err := strconv.ParseInt(line[3], 10, 32)
			if err != nil {
				return fmt.Errorf("cannot parse qty '%s' in txns: %s", line[3], err.Error())
			}
			price, err := strconv.ParseFloat(line[4], 64)
			if err != nil {
				return fmt.Errorf("cannot parse price '%s' in txns: %s", line[4], err.Error())
			}

			if fParquet == nil {
				curFilePath = fmt.Sprintf("%s_%03d.parquet", fileInTxnsPath, fileCounter)
				fParquet, err = os.Create(curFilePath)
				if err != nil {
					return fmt.Errorf("cannot create file '%s': %s", curFilePath, err.Error())
				}

				fileCounter++
				newElCounter = 0

				w, err = storage.NewParquetWriter(fParquet, sc.ParquetCodecGzip)
				if err != nil {
					return err
				}
				if err := w.AddColumn("ts", sc.FieldTypeDateTime); err != nil {
					return err
				}
				if err := w.AddColumn("account_id", sc.FieldTypeString); err != nil {
					return err
				}
				if err := w.AddColumn("ticker", sc.FieldTypeString); err != nil {
					return err
				}
				if err := w.AddColumn("qty", sc.FieldTypeInt); err != nil {
					return err
				}
				if err := w.AddColumn("price", sc.FieldTypeFloat); err != nil {
					return err
				}
			}

			if err := w.FileWriter.AddData(map[string]any{
				"ts":         storage.ParquetWriterMilliTs(d),
				"account_id": accId,
				"ticker":     line[2],
				"qty":        qty,
				"price":      price,
			}); err != nil {
				return fmt.Errorf("cannot write '%s' to txns: %s", accId, err.Error())
			}

			newElCounter++

			if newElCounter == splitCount {
				if err := w.Close(); err != nil {
					return fmt.Errorf("cannot close parquet writer '%s': %s", curFilePath, err.Error())
				}

				if err := fParquet.Close(); err != nil {
					return fmt.Errorf("cannot close file '%s': %s", curFilePath, err.Error())
				}

				fParquet = nil
				w = nil
			}
		}
	}

	if fParquet != nil {
		if err := w.Close(); err != nil {
			return fmt.Errorf("cannot close parquet writer '%s': %s", fileInTxnsPath, err.Error())
		}

		if err := fParquet.Close(); err != nil {
			return fmt.Errorf("cannot close file '%s': %s", fileInTxnsPath, err.Error())
		}
	}

	return nil
}

func generateOutTotals(fileQuickAccountYearPath string, fileOutAccountYearPath string, bigAccountsMap map[string][]string) error {
	f, err := os.Open(fileQuickAccountYearPath)
	if err != nil {
		return fmt.Errorf("cannot open %s: %s", fileQuickAccountYearPath, err.Error())
	}

	fParquet, err := os.Create(fileOutAccountYearPath)
	if err != nil {
		return fmt.Errorf("cannot create file '%s': %s", fileOutAccountYearPath, err.Error())
	}

	w, err := storage.NewParquetWriter(fParquet, sc.ParquetCodecGzip)
	if err != nil {
		return err
	}
	if err := w.AddColumn("ARK fund", sc.FieldTypeString); err != nil {
		return err
	}
	if err := w.AddColumn("Period", sc.FieldTypeString); err != nil {
		return err
	}
	if err := w.AddColumn("Sector", sc.FieldTypeString); err != nil {
		return err
	}
	if err := w.AddColumn("Time-weighted annualized return %", sc.FieldTypeFloat); err != nil {
		return err
	}

	r := csv.NewReader(bufio.NewReader(f))
	isHeader := true
	for {
		line, err := r.Read()
		if err == io.EOF {
			break
		}
		if isHeader {
			isHeader = false
			continue
		}

		accPrefix := line[0]
		accIds, ok := bigAccountsMap[accPrefix]
		if !ok {
			return fmt.Errorf("unknown account prefix '%s' in account_year_perf_baseline", accPrefix)
		}
		for _, accId := range accIds {
			ret, err := strconv.ParseFloat(line[3], 64)
			if err != nil {
				return fmt.Errorf("cannot parse ret '%s' in account_year_perf_baseline: %s", line[3], err.Error())
			}

			if err := w.FileWriter.AddData(map[string]any{
				"ARK fund":                          accId,
				"Period":                            line[1],
				"Sector":                            line[2],
				"Time-weighted annualized return %": ret,
			}); err != nil {
				return fmt.Errorf("cannot write '%s' to account_year_perf_baseline: %s", accId, err.Error())
			}
		}
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("cannot close parquet writer '%s': %s", fileOutAccountYearPath, err.Error())
	}

	if err := fParquet.Close(); err != nil {
		return fmt.Errorf("cannot close file '%s': %s", fileOutAccountYearPath, err.Error())
	}

	return nil
}

func generateOutBySector(fileQuickAccountPeriodSectorPath string, fileOutAccountPeriodSectorPath string, bigAccountsMap map[string][]string) error {
	f, err := os.Open(fileQuickAccountPeriodSectorPath)
	if err != nil {
		return fmt.Errorf("cannot open %s: %s", fileQuickAccountPeriodSectorPath, err.Error())
	}

	fParquet, err := os.Create(fileOutAccountPeriodSectorPath)
	if err != nil {
		return fmt.Errorf("cannot create file '%s': %s", fileOutAccountPeriodSectorPath, err.Error())
	}

	w, err := storage.NewParquetWriter(fParquet, sc.ParquetCodecGzip)
	if err != nil {
		return err
	}
	if err := w.AddColumn("ARK fund", sc.FieldTypeString); err != nil {
		return err
	}
	if err := w.AddColumn("Period", sc.FieldTypeString); err != nil {
		return err
	}
	if err := w.AddColumn("Sector", sc.FieldTypeString); err != nil {
		return err
	}
	if err := w.AddColumn("Time-weighted annualized return %", sc.FieldTypeFloat); err != nil {
		return err
	}

	r := csv.NewReader(bufio.NewReader(f))
	isHeader := true
	for {
		line, err := r.Read()
		if err == io.EOF {
			break
		}
		if isHeader {
			isHeader = false
			continue
		}

		accPrefix := line[0]
		accIds, ok := bigAccountsMap[accPrefix]
		if !ok {
			return fmt.Errorf("unknown account prefix '%s' in account_period_sector_perf_baseline", accPrefix)
		}
		for _, accId := range accIds {
			ret, err := strconv.ParseFloat(line[3], 64)
			if err != nil {
				return fmt.Errorf("cannot parse ret '%s' in account_period_sector_perf_baseline: %s", line[3], err.Error())
			}

			if err := w.FileWriter.AddData(map[string]any{
				"ARK fund":                          accId,
				"Period":                            line[1],
				"Sector":                            line[2],
				"Time-weighted annualized return %": ret,
			}); err != nil {
				return fmt.Errorf("cannot write '%s' to account_period_sector_perf_baseline: %s", accId, err.Error())
			}
		}
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("cannot close parquet writer '%s': %s", fileOutAccountPeriodSectorPath, err.Error())
	}

	if err := fParquet.Close(); err != nil {
		return fmt.Errorf("cannot close file '%s': %s", fileOutAccountPeriodSectorPath, err.Error())
	}

	return nil
}

func generateAccounts(fileInAccountsPath string, quickAccountsMap map[string]string, bigAccountsMap map[string][]string) error {
	fParquet, err := os.Create(fileInAccountsPath + ".parquet")
	if err != nil {
		return fmt.Errorf("cannot create file '%s': %s", fileInAccountsPath, err.Error())
	}

	w, err := storage.NewParquetWriter(fParquet, sc.ParquetCodecGzip)
	if err != nil {
		return err
	}

	if err := w.AddColumn("account_id", sc.FieldTypeString); err != nil {
		return err
	}
	if err := w.AddColumn("earliest_period_start", sc.FieldTypeDateTime); err != nil {
		return err
	}

	for accPrefix, accIds := range bigAccountsMap {
		for _, accId := range accIds {
			eps, ok := quickAccountsMap[accPrefix]
			if !ok {
				return fmt.Errorf("cannot find account prefix '%s' in accounts", accPrefix)
			}
			d, err := time.Parse("2006-01-02", eps)
			if err != nil {
				return fmt.Errorf("cannot parse account earliest_period_start '%s': %s", eps, err.Error())
			}
			if err := w.FileWriter.AddData(map[string]any{
				"account_id":            accId,
				"earliest_period_start": storage.ParquetWriterMilliTs(d),
			}); err != nil {
				return fmt.Errorf("cannot write '%s,%v' to accounts: %s", accId, d, err.Error())
			}
		}
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("cannot close parquet writer '%s': %s", fileInAccountsPath, err.Error())
	}

	if err := fParquet.Close(); err != nil {
		return fmt.Errorf("cannot close file '%s': %s", fileInAccountsPath, err.Error())
	}

	return nil
}

const SOURCE_TXNS int = 88459
const SOURCE_HOLDINGS int = 4300
const HOLDING_FILES_TOTAL int = 10
const TXN_FILES_TOTAL int = 500 // For perf testing purposes, this better be > total_number_of_daemon_cores

func main() {
	quicktestIn := flag.String("quicktest_in", "../../../data/in/portfolio_quicktest", "Root dir for in quicktest files to be used as a template")
	quicktestOut := flag.String("quicktest_out", "../../../data/out/portfolio_quicktest", "Root dir for out quicktest files to be used as a template")
	inRoot := flag.String("in_root", "/tmp/capi_in/portfolio_bigtest", "Root dir for generated in files")
	outRoot := flag.String("out_root", "/tmp/capi_out/portfolio_bigtest", "Root dir for generated out files")
	totalAccountsSuggested := flag.Int("accounts", 100, "Total number of accounts to generate")
	flag.Parse()

	// Template files
	fileQuickAccountsPath := *quicktestIn + "/accounts.csv"
	fileQuickHoldingsPath := *quicktestIn + "/holdings.csv"
	fileQuickTxnsPath := *quicktestIn + "/txns.csv"
	fileQuickAccountYearPath := *quicktestOut + "/account_year_perf_baseline.csv"
	fileQuickAccountPeriodSectorPath := *quicktestOut + "/account_period_sector_perf_baseline.csv"

	// Files to generate
	fileInAccountsPath := *inRoot + "/accounts"
	fileInHoldingsPath := *inRoot + "/holdings"
	fileInTxnsPath := *inRoot + "/txns"
	fileOutAccountYearPath := *outRoot + "/account_year_perf_baseline.parquet"
	fileOutAccountPeriodSectorPath := *outRoot + "/account_period_sector_perf_baseline.parquet"

	// Map to new acct ids

	quickAccountsMap, quickAccounts, err := readQuickAccounts(fileQuickAccountsPath)
	if err != nil {
		log.Fatal(err)
	}

	accountsPerOriginalQuick := *totalAccountsSuggested / len(quickAccounts)
	totalAccounts := accountsPerOriginalQuick * len(quickAccounts)

	fmt.Printf("Generating %d accounts, %d txn files, total %d txns to %s and %s...\n",
		totalAccounts, TXN_FILES_TOTAL, accountsPerOriginalQuick*SOURCE_TXNS, *inRoot, *outRoot)

	bigAccountsMap := map[string][]string{} // ARKK-> [ARKK-000000,ARKK-000001]
	for i := 0; i < totalAccounts; i++ {
		accLocalIdx := i / len(quickAccounts) //0,0,0,0,0,0,1,1,1,1,1,1,2,2,2,2,2,2,
		accPrefix := quickAccounts[i%len(quickAccounts)]
		if _, ok := bigAccountsMap[accPrefix]; !ok {
			bigAccountsMap[accPrefix] = make([]string, accountsPerOriginalQuick)
		}
		bigAccountsMap[accPrefix][accLocalIdx] = fmt.Sprintf("%s-%06d", accPrefix, i)
	}

	// Accounts

	if err := generateAccounts(fileInAccountsPath, quickAccountsMap, bigAccountsMap); err != nil {
		log.Fatal(err)
	}

	// Holdings

	if err := generateHoldings(fileQuickHoldingsPath, fileInHoldingsPath, bigAccountsMap, SOURCE_HOLDINGS*accountsPerOriginalQuick/HOLDING_FILES_TOTAL+1); err != nil {
		log.Fatal(err)
	}

	// Txns

	if err := generateTxns(fileQuickTxnsPath, fileInTxnsPath, bigAccountsMap, SOURCE_TXNS*accountsPerOriginalQuick/TXN_FILES_TOTAL+1); err != nil {
		log.Fatal(err)
	}

	// Out totals

	if err := generateOutTotals(fileQuickAccountYearPath, fileOutAccountYearPath, bigAccountsMap); err != nil {
		log.Fatal(err)
	}

	// Out by sector

	if err := generateOutBySector(fileQuickAccountPeriodSectorPath, fileOutAccountPeriodSectorPath, bigAccountsMap); err != nil {
		log.Fatal(err)
	}

}
