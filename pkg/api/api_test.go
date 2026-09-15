package api

/*
These are "unit" tests that use gocqlmem and exercise ProcessDataBatchMsg implementation.
They all require copy_demo_date.sh to copy test data.
They take between 3 an 18 seconds each - this is why we do not include them in test_unit.sh
They test db-level Cassandra conditions, hence the somewhat comlex  TableInserterQueryPerformer interface.
*/
import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/capillariesio/capillaries/pkg/cql"
	"github.com/capillariesio/capillaries/pkg/ctx"
	"github.com/capillariesio/capillaries/pkg/custom/pycalc"
	"github.com/capillariesio/capillaries/pkg/custom/taganddenormalize"
	"github.com/capillariesio/capillaries/pkg/db"
	"github.com/capillariesio/capillaries/pkg/env"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/mq"
	"github.com/capillariesio/capillaries/pkg/sc"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
	"github.com/capillariesio/gocqlmem/gocqlshims"
	"github.com/stretchr/testify/assert"
)

func readCSV(filename string) ([][]string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)

	var rows [][]string
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%s: %w", filename, err)
		}
		rows = append(rows, row)
	}

	return rows, nil
}

func compareCsvs(file1 string, file2 string) error {
	a, err := readCSV(file1)
	if err != nil {
		return err
	}

	b, err := readCSV(file2)
	if err != nil {
		return err
	}

	maxRows := len(a)
	if len(b) > maxRows {
		maxRows = len(b)
	}

	for i := 0; i < maxRows; i++ {
		// Row exists only in a.csv.
		if i >= len(b) {
			return fmt.Errorf("Row %d: missing in the second csv: %q", i+1, a[i])
		}

		// Row exists only in b.csv.
		if i >= len(a) {
			return fmt.Errorf("Row %d: extra in the second csv: %q", i+1, b[i])
		}

		maxFields := len(a[i])
		if len(b[i]) > maxFields {
			maxFields = len(b[i])
		}

		for j := 0; j < maxFields; j++ {
			var av, bv string

			if j < len(a[i]) {
				av = a[i][j]
			}
			if j < len(b[i]) {
				bv = b[i][j]
			}

			if av != bv {
				return fmt.Errorf("Row %d, column %d: a.csv=%q, b.csv=%q", i+1, j+1, av, bv)
			}
		}
	}

	return nil
}

type TestProcessorDefFactory struct {
}

func (f *TestProcessorDefFactory) Create(processorType string) (sc.CustomProcessorDef, bool) {
	switch processorType {
	case pycalc.ProcessorPyCalcName:
		return &pycalc.PyCalcProcessorDef{}, true
	case taganddenormalize.ProcessorTagAndDenormalizeName:
		return &taganddenormalize.TagAndDenormalizeProcessorDef{}, true
	default:
		return nil, false
	}
}

func getTestProcessorSettings() map[string]json.RawMessage {
	jsonData := []byte(`{"py_calc": {"python_interpreter_path": "python", "python_interpreter_params": ["-u", "-"]}, "tag_and_denormalize": {}}`)
	var result map[string]json.RawMessage
	err := json.Unmarshal(jsonData, &result)
	if err != nil {
		panic(err)
	}
	return result
}

// Table does not exist

type TableInserterQueryPerformerTestDataAndIdxDoesNotExist struct {
	TotalDataHits           int
	TotalIdxHits            int
	SimulatedDataErrorCount int
	SimulatedIdxErrorCount  int
}

func (qp *TableInserterQueryPerformerTestDataAndIdxDoesNotExist) PerformInsertDataRecordWithRowid(gocqlSession gocqlshims.Session, pq *cql.PreparedQuery, preparedDataQueryParams []any, retryCount int) (map[string]any, bool, error) {
	qp.TotalDataHits++
	if retryCount > 0 || qp.SimulatedDataErrorCount >= 1 {
		return db.HelperPerformInsertDataRecordWithRowid(gocqlSession, pq, preparedDataQueryParams)
	}

	qp.SimulatedDataErrorCount++
	// log: will wait for table ... to be created, table retry count 0, got does not exist
	// retry and succeed
	return nil, false, errors.New("test scenario: data table " + cql.ErrorDoesNotExist)
}

func (qp *TableInserterQueryPerformerTestDataAndIdxDoesNotExist) PerformInsertIdxRecordWithRowid(idxUniqueness sc.IdxUniqueness, gocqlSession gocqlshims.Session, pq *cql.PreparedQuery, preparedIdxQueryParams []any, retryCount int) (map[string]any, int, bool, error) {
	qp.TotalIdxHits++
	if retryCount > 0 || qp.SimulatedIdxErrorCount >= 1 {
		return db.HelperPerformInsertIdxRecordWithRowid(idxUniqueness, gocqlSession, pq, preparedIdxQueryParams, retryCount)
	}

	qp.SimulatedIdxErrorCount++
	// log: will wait for idx table ... to be created, table retry count 0, got does not exist
	// retry and succeed
	return nil, retryCount, false, errors.New("test scenario: idx table " + cql.ErrorDoesNotExist)
}

func TestTableDoesNotExistLookup(t *testing.T) {
	os.Remove("/tmp/capi_out/lookup_quicktest/order_date_value_grouped_inner.csv")
	os.Remove("/tmp/capi_out/lookup_quicktest/order_date_value_grouped_left_outer.csv")
	os.Remove("/tmp/capi_out/lookup_quicktest/order_item_date_inner.csv")
	os.Remove("/tmp/capi_out/lookup_quicktest/order_item_date_left_outer.csv")

	ks := "ks_table_does_not_exist_lookup"

	envConfig := env.EnvConfig{
		Cassandra:                         env.CassandraConfig{WriterWorkers: 2},
		Log:                               env.LogConfig{Level: "INFO"},
		CustomProcessorDefFactoryInstance: &TestProcessorDefFactory{},
		CustomProcessorsSettings:          getTestProcessorSettings(),
		UseGocqlmem:                       true,
	}
	sc.ScriptDefCache = sc.NewScriptDefCache()
	NodeDependencyReadynessCache = NewNodeDependencyReadynessCache()

	logger, err := l.NewLoggerFromEnvConfig(&envConfig, "unittest")
	assert.Nil(t, err)
	logger.PushF("TestTableDoesNotExistFannieMae")
	defer logger.PopF()

	mqProducer := mq.TestInmemProducer{}

	gocqlmemSession, cassandraEngineType, err := db.NewSession(&envConfig, ks, db.CreateKeyspaceOnConnect)
	assert.Nil(t, err)

	_, err = StartRun(&envConfig, logger, &mqProducer, "/tmp/capi_cfg/lookup_quicktest/script_quick.yaml", "/tmp/capi_cfg/lookup_quicktest/script_params_quick_fs_one.yaml", gocqlmemSession, cassandraEngineType, ks, []string{"read_orders", "read_order_items"}, "test run")
	assert.Nil(t, err)

	var runStatus wfmodel.RunStatusType

	// Verify run status
	runHistory, err := GetRunHistory(gocqlmemSession, ks)
	assert.Nil(t, err)
	runStatus = runHistory[len(runHistory)-1].Status
	assert.Equal(t, wfmodel.RunStart, runStatus)

	for {
		msg := mqProducer.PeekHead()
		if msg == nil {
			break
		}
		queryPerformer := TableInserterQueryPerformerTestDataAndIdxDoesNotExist{}
		ackCmd := ProcessDataBatchMsg(&envConfig, logger, msg, 0, nil, ctx.TableInserterProperties{
			QueryPerformer:               &queryPerformer,
			DoesNotExistPauseMillis:      10, // speed it up for testing
			OperationTimedOutPauseMillis: 10, // speed it up for testing
		})
		if ackCmd == mq.AcknowledgerCmdAck {
			mqProducer.RemoveHead()
		} else {
			mqProducer.MoveHeadToTail()
		}

		// Make sure that fake errors were actually introduced
		if queryPerformer.TotalDataHits > 0 {
			if strings.HasPrefix(msg.TargetNodeName, "file_") {
				assert.Equal(t, 0, queryPerformer.SimulatedDataErrorCount, msg.TargetNodeName)
			} else {
				assert.Equal(t, 1, queryPerformer.SimulatedDataErrorCount, msg.TargetNodeName)
			}
		}

		if queryPerformer.TotalIdxHits > 0 {
			if strings.HasPrefix(msg.TargetNodeName, "read_") {
				assert.Equal(t, 1, queryPerformer.SimulatedIdxErrorCount, msg.TargetNodeName)
			} else {
				assert.Equal(t, 0, queryPerformer.SimulatedIdxErrorCount, msg.TargetNodeName)
			}
		}
	}

	// Verify run status
	runHistory, err = GetRunHistory(gocqlmemSession, ks)
	assert.Nil(t, err)
	runStatus = runHistory[len(runHistory)-1].Status
	assert.Equal(t, wfmodel.RunComplete, runStatus)

	err = compareCsvs("/tmp/capi_out/lookup_quicktest/order_date_value_grouped_inner_baseline.csv", "/tmp/capi_out/lookup_quicktest/order_date_value_grouped_inner.csv")
	assert.Nil(t, err)
	err = compareCsvs("/tmp/capi_out/lookup_quicktest/order_date_value_grouped_left_outer_baseline.csv", "/tmp/capi_out/lookup_quicktest/order_date_value_grouped_left_outer.csv")
	assert.Nil(t, err)
	err = compareCsvs("/tmp/capi_out/lookup_quicktest/order_item_date_inner_baseline.csv", "/tmp/capi_out/lookup_quicktest/order_item_date_inner.csv")
	assert.Nil(t, err)
	err = compareCsvs("/tmp/capi_out/lookup_quicktest/order_item_date_left_outer_baseline.csv", "/tmp/capi_out/lookup_quicktest/order_item_date_left_outer.csv")
	assert.Nil(t, err)

	assert.Nil(t, gocqlmemSession.Query(fmt.Sprintf("DROP keyspace %s;", ks)).Exec())
	gocqlmemSession.Close()
}

func TestTableDoesNotExistFannieMae(t *testing.T) {
	os.Remove("/tmp/capi_out/fannie_mae_apitest/deal_seller_summaries.csv")
	os.Remove("/tmp/capi_out/fannie_mae_apitest/deal_summaries.csv")
	os.Remove("/tmp/capi_out/fannie_mae_apitest/loan_smrs_clcltd.csv")

	ks := "ks_table_does_not_exist_fannie_mae"

	envConfig := env.EnvConfig{
		Cassandra:                         env.CassandraConfig{WriterWorkers: 4},
		Log:                               env.LogConfig{Level: "INFO"},
		CustomProcessorDefFactoryInstance: &TestProcessorDefFactory{},
		CustomProcessorsSettings:          getTestProcessorSettings(),
		UseGocqlmem:                       true,
	}
	sc.ScriptDefCache = sc.NewScriptDefCache()
	NodeDependencyReadynessCache = NewNodeDependencyReadynessCache()

	logger, err := l.NewLoggerFromEnvConfig(&envConfig, "unittest")
	assert.Nil(t, err)
	logger.PushF("TestTableDoesNotExistFannieMae")
	defer logger.PopF()

	mqProducer := mq.TestInmemProducer{}

	gocqlmemSession, cassandraEngineType, err := db.NewSession(&envConfig, ks, db.CreateKeyspaceOnConnect)
	assert.Nil(t, err)

	_, err = StartRun(&envConfig, logger, &mqProducer, "/tmp/capi_cfg/fannie_mae_apitest/script_api.json", "/tmp/capi_cfg/fannie_mae_apitest/script_params_api.json", gocqlmemSession, cassandraEngineType, ks, []string{"01_read_payments"}, "test run")
	assert.Nil(t, err)

	var runStatus wfmodel.RunStatusType

	// Verify run status
	runHistory, err := GetRunHistory(gocqlmemSession, ks)
	assert.Nil(t, err)
	runStatus = runHistory[len(runHistory)-1].Status
	assert.Equal(t, wfmodel.RunStart, runStatus)

	for {
		msg := mqProducer.PeekHead()
		if msg == nil {
			break
		}
		queryPerformer := TableInserterQueryPerformerTestDataAndIdxDoesNotExist{}
		ackCmd := ProcessDataBatchMsg(&envConfig, logger, msg, 0, nil, ctx.TableInserterProperties{
			QueryPerformer:               &queryPerformer,
			DoesNotExistPauseMillis:      10,  // speed it up for testing
			OperationTimedOutPauseMillis: 100, // speed it up for testing
		})
		if ackCmd == mq.AcknowledgerCmdAck {
			mqProducer.RemoveHead()
			logger.Info("batch processed %s", msg.FullBatchId())
		} else {
			mqProducer.MoveHeadToTail()
		}

		// Make sure that fake errors were actually introduced
		if queryPerformer.TotalDataHits > 0 {
			if strings.Contains(msg.TargetNodeName, "write_file_") {
				assert.Equal(t, 0, queryPerformer.SimulatedDataErrorCount, msg.TargetNodeName)
			} else {
				assert.Equal(t, 1, queryPerformer.SimulatedDataErrorCount, msg.TargetNodeName)
			}
		}

		if queryPerformer.TotalIdxHits > 0 {
			if strings.HasPrefix(msg.TargetNodeName, "01_") || strings.HasPrefix(msg.TargetNodeName, "02_") || msg.TargetNodeName == "04_loan_smrs_clcltd" {
				assert.Equal(t, 1, queryPerformer.SimulatedIdxErrorCount, msg.TargetNodeName)
			} else {
				assert.Equal(t, 0, queryPerformer.SimulatedIdxErrorCount, msg.TargetNodeName)
			}
		}
	}

	// Verify run status
	runHistory, err = GetRunHistory(gocqlmemSession, ks)
	assert.Nil(t, err)
	runStatus = runHistory[len(runHistory)-1].Status
	assert.Equal(t, wfmodel.RunComplete, runStatus)

	err = compareCsvs("/tmp/capi_out/fannie_mae_apitest/deal_seller_summaries_baseline.csv", "/tmp/capi_out/fannie_mae_apitest/deal_seller_summaries.csv")
	assert.Nil(t, err)
	err = compareCsvs("/tmp/capi_out/fannie_mae_apitest/deal_summaries_baseline.csv", "/tmp/capi_out/fannie_mae_apitest/deal_summaries.csv")
	assert.Nil(t, err)
	err = compareCsvs("/tmp/capi_out/fannie_mae_apitest/loan_smrs_clcltd_baseline.csv", "/tmp/capi_out/fannie_mae_apitest/loan_smrs_clcltd.csv")
	assert.Nil(t, err)

	assert.Nil(t, gocqlmemSession.Query(fmt.Sprintf("DROP keyspace %s;", ks)).Exec())
	gocqlmemSession.Close()
}

// OperationTimedOut

type TableInserterQueryPerformerTestOperationTimedOut struct {
	TotalDataHits           int
	TotalIdxHits            int
	SimulatedDataErrorCount int
	SimulatedIdxErrorCount  int
}

func (qp *TableInserterQueryPerformerTestOperationTimedOut) PerformInsertDataRecordWithRowid(gocqlSession gocqlshims.Session, pq *cql.PreparedQuery, preparedDataQueryParams []any, retryCount int) (map[string]any, bool, error) {
	qp.TotalDataHits++
	if retryCount > 0 || qp.SimulatedDataErrorCount >= 1 {
		return db.HelperPerformInsertDataRecordWithRowid(gocqlSession, pq, preparedDataQueryParams)
	}

	qp.SimulatedDataErrorCount++
	// log: cluster overloaded (Operation timed out), will wait for ...ms before writing to data table ... again, table retry count 0
	// retry and succeed
	return nil, false, errors.New("test scenario: data table " + cql.ErrorOperationTimedOut)
}

func (qp *TableInserterQueryPerformerTestOperationTimedOut) PerformInsertIdxRecordWithRowid(idxUniqueness sc.IdxUniqueness, gocqlSession gocqlshims.Session, pq *cql.PreparedQuery, preparedIdxQueryParams []any, retryCount int) (map[string]any, int, bool, error) {
	qp.TotalIdxHits++
	if retryCount > 0 || qp.SimulatedIdxErrorCount >= 1 {
		return db.HelperPerformInsertIdxRecordWithRowid(idxUniqueness, gocqlSession, pq, preparedIdxQueryParams, retryCount)
	}

	qp.SimulatedIdxErrorCount++
	// log: cluster overloaded (Operation timed out), will wait for ...ms before writing to data table ... again, table retry count 0
	// retry and succeed
	return nil, retryCount, false, errors.New("test scenario: idx table " + cql.ErrorOperationTimedOut)
}

func TestOperationTimedOutLookup(t *testing.T) {
	os.Remove("/tmp/capi_out/lookup_quicktest/order_date_value_grouped_inner.csv")
	os.Remove("/tmp/capi_out/lookup_quicktest/order_date_value_grouped_left_outer.csv")
	os.Remove("/tmp/capi_out/lookup_quicktest/order_item_date_inner.csv")
	os.Remove("/tmp/capi_out/lookup_quicktest/order_item_date_left_outer.csv")

	ks := "ks_operation_timed_out_lookup"

	envConfig := env.EnvConfig{
		Cassandra:                         env.CassandraConfig{WriterWorkers: 2},
		Log:                               env.LogConfig{Level: "INFO"},
		CustomProcessorDefFactoryInstance: &TestProcessorDefFactory{},
		CustomProcessorsSettings:          getTestProcessorSettings(),
		UseGocqlmem:                       true,
	}
	sc.ScriptDefCache = sc.NewScriptDefCache()
	NodeDependencyReadynessCache = NewNodeDependencyReadynessCache()

	logger, err := l.NewLoggerFromEnvConfig(&envConfig, "unittest")
	assert.Nil(t, err)
	logger.PushF("TestOperationTimedOutLookup")
	defer logger.PopF()

	mqProducer := mq.TestInmemProducer{}

	gocqlmemSession, cassandraEngineType, err := db.NewSession(&envConfig, ks, db.CreateKeyspaceOnConnect)
	assert.Nil(t, err)

	_, err = StartRun(&envConfig, logger, &mqProducer, "/tmp/capi_cfg/lookup_quicktest/script_quick.yaml", "/tmp/capi_cfg/lookup_quicktest/script_params_quick_fs_one.yaml", gocqlmemSession, cassandraEngineType, ks, []string{"read_orders", "read_order_items"}, "test run")
	assert.Nil(t, err)

	var runStatus wfmodel.RunStatusType

	// Verify run status
	runHistory, err := GetRunHistory(gocqlmemSession, ks)
	assert.Nil(t, err)
	runStatus = runHistory[len(runHistory)-1].Status
	assert.Equal(t, wfmodel.RunStart, runStatus)

	for {
		msg := mqProducer.PeekHead()
		if msg == nil {
			break
		}
		queryPerformer := TableInserterQueryPerformerTestOperationTimedOut{}
		ackCmd := ProcessDataBatchMsg(&envConfig, logger, msg, 0, nil, ctx.TableInserterProperties{
			QueryPerformer:               &queryPerformer,
			DoesNotExistPauseMillis:      10, // speed it up for testing
			OperationTimedOutPauseMillis: 10, // speed it up for testing
		})
		if ackCmd == mq.AcknowledgerCmdAck {
			mqProducer.RemoveHead()
		} else {
			mqProducer.MoveHeadToTail()
		}

		// Make sure that fake errors were actually introduced
		if queryPerformer.TotalDataHits > 0 {
			if strings.Contains(msg.TargetNodeName, "write_file_") {
				assert.Equal(t, 0, queryPerformer.SimulatedDataErrorCount, msg.TargetNodeName)
			} else {
				assert.Equal(t, 1, queryPerformer.SimulatedDataErrorCount, msg.TargetNodeName)
			}
		}

		if queryPerformer.TotalIdxHits > 0 {
			if strings.HasPrefix(msg.TargetNodeName, "read_") {
				assert.Equal(t, 1, queryPerformer.SimulatedIdxErrorCount, msg.TargetNodeName)
			} else {
				assert.Equal(t, 0, queryPerformer.SimulatedIdxErrorCount, msg.TargetNodeName)
			}
		}
	}

	// Verify run status
	runHistory, err = GetRunHistory(gocqlmemSession, ks)
	assert.Nil(t, err)
	runStatus = runHistory[len(runHistory)-1].Status
	assert.Equal(t, wfmodel.RunComplete, runStatus)

	err = compareCsvs("/tmp/capi_out/lookup_quicktest/order_date_value_grouped_inner_baseline.csv", "/tmp/capi_out/lookup_quicktest/order_date_value_grouped_inner.csv")
	assert.Nil(t, err)
	err = compareCsvs("/tmp/capi_out/lookup_quicktest/order_date_value_grouped_left_outer_baseline.csv", "/tmp/capi_out/lookup_quicktest/order_date_value_grouped_left_outer.csv")
	assert.Nil(t, err)
	err = compareCsvs("/tmp/capi_out/lookup_quicktest/order_item_date_inner_baseline.csv", "/tmp/capi_out/lookup_quicktest/order_item_date_inner.csv")
	assert.Nil(t, err)
	err = compareCsvs("/tmp/capi_out/lookup_quicktest/order_item_date_left_outer_baseline.csv", "/tmp/capi_out/lookup_quicktest/order_item_date_left_outer.csv")
	assert.Nil(t, err)

	assert.Nil(t, gocqlmemSession.Query(fmt.Sprintf("DROP keyspace %s;", ks)).Exec())
	gocqlmemSession.Close()
}

func TestOperationTimedOutFannieMae(t *testing.T) {
	os.Remove("/tmp/capi_out/fannie_mae_apitest/deal_seller_summaries.csv")
	os.Remove("/tmp/capi_out/fannie_mae_apitest/deal_summaries.csv")
	os.Remove("/tmp/capi_out/fannie_mae_apitest/loan_smrs_clcltd.csv")

	ks := "ks_operation_timed_out_fannie_mae"

	envConfig := env.EnvConfig{
		Cassandra:                         env.CassandraConfig{WriterWorkers: 2},
		Log:                               env.LogConfig{Level: "INFO"},
		CustomProcessorDefFactoryInstance: &TestProcessorDefFactory{},
		CustomProcessorsSettings:          getTestProcessorSettings(),
		UseGocqlmem:                       true,
	}
	sc.ScriptDefCache = sc.NewScriptDefCache()
	NodeDependencyReadynessCache = NewNodeDependencyReadynessCache()

	logger, err := l.NewLoggerFromEnvConfig(&envConfig, "unittest")
	assert.Nil(t, err)
	logger.PushF("TestOperationTimedOutFannieMae")
	defer logger.PopF()

	mqProducer := mq.TestInmemProducer{}

	gocqlmemSession, cassandraEngineType, err := db.NewSession(&envConfig, ks, db.CreateKeyspaceOnConnect)
	assert.Nil(t, err)

	_, err = StartRun(&envConfig, logger, &mqProducer, "/tmp/capi_cfg/fannie_mae_apitest/script_api.json", "/tmp/capi_cfg/fannie_mae_apitest/script_params_api.json", gocqlmemSession, cassandraEngineType, ks, []string{"01_read_payments"}, "test run")
	assert.Nil(t, err)

	var runStatus wfmodel.RunStatusType

	// Verify run status
	runHistory, err := GetRunHistory(gocqlmemSession, ks)
	assert.Nil(t, err)
	runStatus = runHistory[len(runHistory)-1].Status
	assert.Equal(t, wfmodel.RunStart, runStatus)

	for {
		msg := mqProducer.PeekHead()
		if msg == nil {
			break
		}
		queryPerformer := TableInserterQueryPerformerTestOperationTimedOut{}
		ackCmd := ProcessDataBatchMsg(&envConfig, logger, msg, 0, nil, ctx.TableInserterProperties{
			QueryPerformer:               &queryPerformer,
			DoesNotExistPauseMillis:      10,  // speed it up for testing
			OperationTimedOutPauseMillis: 100, // speed it up for testing
		})
		if ackCmd == mq.AcknowledgerCmdAck {
			mqProducer.RemoveHead()
		} else {
			mqProducer.MoveHeadToTail()
		}

		// Make sure that fake errors were actually introduced
		if queryPerformer.TotalDataHits > 0 {
			if strings.Contains(msg.TargetNodeName, "write_file_") {
				assert.Equal(t, 0, queryPerformer.SimulatedDataErrorCount, msg.TargetNodeName)
			} else {
				assert.Equal(t, 1, queryPerformer.SimulatedDataErrorCount, msg.TargetNodeName)
			}
		}

		if queryPerformer.TotalIdxHits > 0 {
			if strings.HasPrefix(msg.TargetNodeName, "01_") || strings.HasPrefix(msg.TargetNodeName, "02_") || msg.TargetNodeName == "04_loan_smrs_clcltd" {
				assert.Equal(t, 1, queryPerformer.SimulatedIdxErrorCount, msg.TargetNodeName)
			} else {
				assert.Equal(t, 0, queryPerformer.SimulatedIdxErrorCount, msg.TargetNodeName)
			}
		}
	}

	// Verify run status
	runHistory, err = GetRunHistory(gocqlmemSession, ks)
	assert.Nil(t, err)
	runStatus = runHistory[len(runHistory)-1].Status
	assert.Equal(t, wfmodel.RunComplete, runStatus)

	err = compareCsvs("/tmp/capi_out/fannie_mae_apitest/deal_seller_summaries_baseline.csv", "/tmp/capi_out/fannie_mae_apitest/deal_seller_summaries.csv")
	assert.Nil(t, err)
	err = compareCsvs("/tmp/capi_out/fannie_mae_apitest/deal_summaries_baseline.csv", "/tmp/capi_out/fannie_mae_apitest/deal_summaries.csv")
	assert.Nil(t, err)
	err = compareCsvs("/tmp/capi_out/fannie_mae_apitest/loan_smrs_clcltd_baseline.csv", "/tmp/capi_out/fannie_mae_apitest/loan_smrs_clcltd.csv")
	assert.Nil(t, err)

	assert.Nil(t, gocqlmemSession.Query(fmt.Sprintf("DROP keyspace %s;", ks)).Exec())
	gocqlmemSession.Close()
}

// Data serious error

type TableInserterQueryPerformerTestDataSeriousError struct {
	TotalDataHits           int
	SimulatedDataErrorCount int
}

func (qp *TableInserterQueryPerformerTestDataSeriousError) PerformInsertDataRecordWithRowid(gocqlSession gocqlshims.Session, pq *cql.PreparedQuery, preparedDataQueryParams []any, retryCount int) (map[string]any, bool, error) {
	qp.TotalDataHits++
	if retryCount > 0 || qp.SimulatedDataErrorCount >= 1 {
		return db.HelperPerformInsertDataRecordWithRowid(gocqlSession, pq, preparedDataQueryParams)
	}

	qp.SimulatedDataErrorCount++
	// log: some serious error; cannot write to data table
	// give up immediately and report failure
	return nil, false, errors.New("test scenario: data table " + cql.ErrorSomeSeriousError)
}

func (qp *TableInserterQueryPerformerTestDataSeriousError) PerformInsertIdxRecordWithRowid(idxUniqueness sc.IdxUniqueness, gocqlSession gocqlshims.Session, pq *cql.PreparedQuery, preparedIdxQueryParams []any, retryCount int) (map[string]any, int, bool, error) {
	return db.HelperPerformInsertIdxRecordWithRowid(idxUniqueness, gocqlSession, pq, preparedIdxQueryParams, retryCount)
}

func TestDataSeriousErrorLookup(t *testing.T) {
	os.Remove("/tmp/capi_out/lookup_quicktest/order_date_value_grouped_inner.csv")
	os.Remove("/tmp/capi_out/lookup_quicktest/order_date_value_grouped_left_outer.csv")
	os.Remove("/tmp/capi_out/lookup_quicktest/order_item_date_inner.csv")
	os.Remove("/tmp/capi_out/lookup_quicktest/order_item_date_left_outer.csv")

	ks := "ks_data_serious_error_lookup"

	envConfig := env.EnvConfig{
		Cassandra:                         env.CassandraConfig{WriterWorkers: 1},
		Log:                               env.LogConfig{Level: "INFO"},
		CustomProcessorDefFactoryInstance: &TestProcessorDefFactory{},
		CustomProcessorsSettings:          getTestProcessorSettings(),
		UseGocqlmem:                       true,
	}
	sc.ScriptDefCache = sc.NewScriptDefCache()
	NodeDependencyReadynessCache = NewNodeDependencyReadynessCache()

	logger, err := l.NewLoggerFromEnvConfig(&envConfig, "unittest")
	assert.Nil(t, err)
	logger.PushF("TestDataSeriousErrorLookup")
	defer logger.PopF()

	mqProducer := mq.TestInmemProducer{}

	gocqlmemSession, cassandraEngineType, err := db.NewSession(&envConfig, ks, db.CreateKeyspaceOnConnect)
	assert.Nil(t, err)

	_, err = StartRun(&envConfig, logger, &mqProducer, "/tmp/capi_cfg/lookup_quicktest/script_quick.yaml", "/tmp/capi_cfg/lookup_quicktest/script_params_quick_fs_one.yaml", gocqlmemSession, cassandraEngineType, ks, []string{"read_orders", "read_order_items"}, "test run")
	assert.Nil(t, err)

	var runStatus wfmodel.RunStatusType

	// Verify run status
	runHistory, err := GetRunHistory(gocqlmemSession, ks)
	assert.Nil(t, err)
	runStatus = runHistory[len(runHistory)-1].Status
	assert.Equal(t, wfmodel.RunStart, runStatus)

	for {
		msg := mqProducer.PeekHead()
		if msg == nil {
			break
		}
		queryPerformer := TableInserterQueryPerformerTestDataSeriousError{}
		ackCmd := ProcessDataBatchMsg(&envConfig, logger, msg, 0, nil, ctx.TableInserterProperties{
			QueryPerformer:               &queryPerformer,
			DoesNotExistPauseMillis:      10, // speed it up for testing
			OperationTimedOutPauseMillis: 10, // speed it up for testing
		})
		if ackCmd == mq.AcknowledgerCmdAck {
			mqProducer.RemoveHead()
		} else {
			mqProducer.MoveHeadToTail()
		}
	}

	// Verify run status
	runHistory, err = GetRunHistory(gocqlmemSession, ks)
	assert.Nil(t, err)
	runStatus = runHistory[len(runHistory)-1].Status
	assert.Equal(t, wfmodel.RunComplete, runStatus)

	// Verify node statuses
	nodeHistory, err := GetNodeHistoryForRuns(gocqlmemSession, ks, []int16{int16(1)})
	assert.Nil(t, err)
	newNodeRunStatusMap := map[string]wfmodel.NodeBatchStatusType{}
	for _, nodeEvent := range nodeHistory {
		newNodeRunStatusMap[nodeEvent.ScriptNode] = nodeEvent.Status
	}

	// For each node, verify batch statuses
	for nodeName, nodeStatus := range newNodeRunStatusMap {
		assert.Equal(t, wfmodel.NodeBatchFail, nodeStatus, fmt.Sprintf("node %s supposed to fail", nodeName))
		// Make sure all batches for this node started then failed
		batchEvents, err := GetBatchHistoryForRunAndNode(gocqlmemSession, ks, int16(1), nodeName)
		assert.Nil(t, err)
		if nodeName == "read_orders" || nodeName == "read_order_items" {
			// These nodes start and fail
			assert.Equal(t, 2, len(batchEvents), nodeName)
			assert.Equal(t, wfmodel.NodeBatchStart, batchEvents[0].Status, nodeName)
			assert.Equal(t, wfmodel.NodeBatchFail, batchEvents[1].Status, nodeName)
			assert.True(t, strings.Contains(batchEvents[1].Comment, cql.ErrorSomeSeriousError))
		} else {
			// These nodes failed without starting
			for _, event := range batchEvents {
				assert.Equal(t, wfmodel.NodeBatchFail, event.Status, nodeName)
				assert.True(t, strings.Contains(event.Comment, "some dependency nodes"))
			}
		}
	}
	assert.Nil(t, gocqlmemSession.Query(fmt.Sprintf("DROP keyspace %s;", ks)).Exec())
	gocqlmemSession.Close()
}

func TestDataSeriousErrorFannieMae(t *testing.T) {
	os.Remove("/tmp/capi_out/fannie_mae_apitest/deal_seller_summaries.csv")
	os.Remove("/tmp/capi_out/fannie_mae_apitest/deal_summaries.csv")
	os.Remove("/tmp/capi_out/fannie_mae_apitest/loan_smrs_clcltd.csv")

	ks := "ks_data_serious_error_fannie_mae"

	envConfig := env.EnvConfig{
		Cassandra:                         env.CassandraConfig{WriterWorkers: 1},
		Log:                               env.LogConfig{Level: "INFO"},
		CustomProcessorDefFactoryInstance: &TestProcessorDefFactory{},
		CustomProcessorsSettings:          getTestProcessorSettings(),
		UseGocqlmem:                       true,
	}
	sc.ScriptDefCache = sc.NewScriptDefCache()
	NodeDependencyReadynessCache = NewNodeDependencyReadynessCache()

	logger, err := l.NewLoggerFromEnvConfig(&envConfig, "unittest")
	assert.Nil(t, err)
	logger.PushF("TestDataSeriousErrorFannieMae")
	defer logger.PopF()

	mqProducer := mq.TestInmemProducer{}

	gocqlmemSession, cassandraEngineType, err := db.NewSession(&envConfig, ks, db.CreateKeyspaceOnConnect)
	assert.Nil(t, err)

	_, err = StartRun(&envConfig, logger, &mqProducer, "/tmp/capi_cfg/fannie_mae_apitest/script_api.json", "/tmp/capi_cfg/fannie_mae_apitest/script_params_api.json", gocqlmemSession, cassandraEngineType, ks, []string{"01_read_payments"}, "test run")
	assert.Nil(t, err)

	var runStatus wfmodel.RunStatusType

	// Verify run status
	runHistory, err := GetRunHistory(gocqlmemSession, ks)
	assert.Nil(t, err)
	runStatus = runHistory[len(runHistory)-1].Status
	assert.Equal(t, wfmodel.RunStart, runStatus)

	for {
		msg := mqProducer.PeekHead()
		if msg == nil {
			break
		}
		queryPerformer := TableInserterQueryPerformerTestDataSeriousError{}
		ackCmd := ProcessDataBatchMsg(&envConfig, logger, msg, 0, nil, ctx.TableInserterProperties{
			QueryPerformer:               &queryPerformer,
			DoesNotExistPauseMillis:      10,  // speed it up for testing
			OperationTimedOutPauseMillis: 100, // speed it up for testing
		})
		if ackCmd == mq.AcknowledgerCmdAck {
			mqProducer.RemoveHead()
		} else {
			mqProducer.MoveHeadToTail()
		}
	}

	// Verify run status
	runHistory, err = GetRunHistory(gocqlmemSession, ks)
	assert.Nil(t, err)
	runStatus = runHistory[len(runHistory)-1].Status
	assert.Equal(t, wfmodel.RunComplete, runStatus)

	// Verify node statuses
	nodeHistory, err := GetNodeHistoryForRuns(gocqlmemSession, ks, []int16{int16(1)})
	assert.Nil(t, err)
	newNodeRunStatusMap := map[string]wfmodel.NodeBatchStatusType{}
	for _, nodeEvent := range nodeHistory {
		newNodeRunStatusMap[nodeEvent.ScriptNode] = nodeEvent.Status
	}

	// For each node, verify batch statuses
	for nodeName, nodeStatus := range newNodeRunStatusMap {
		assert.Equal(t, wfmodel.NodeBatchFail, nodeStatus, fmt.Sprintf("node %s supposed to fail", nodeName))
		// Make sure all batches for this node started then failed
		batchEvents, err := GetBatchHistoryForRunAndNode(gocqlmemSession, ks, int16(1), nodeName)
		assert.Nil(t, err)
		if nodeName == "01_read_payments" {
			// These nodes start and fail
			assert.Equal(t, 2, len(batchEvents), nodeName)
			assert.Equal(t, wfmodel.NodeBatchStart, batchEvents[0].Status, nodeName)
			assert.Equal(t, wfmodel.NodeBatchFail, batchEvents[1].Status, nodeName)
			assert.True(t, strings.Contains(batchEvents[1].Comment, cql.ErrorSomeSeriousError))
		} else {
			// These nodes failed without starting
			for _, event := range batchEvents {
				assert.Equal(t, wfmodel.NodeBatchFail, event.Status, nodeName)
				assert.True(t, strings.Contains(event.Comment, "some dependency nodes"))
			}
		}
	}
	assert.Nil(t, gocqlmemSession.Query(fmt.Sprintf("DROP keyspace %s;", ks)).Exec())
	gocqlmemSession.Close()
}

// Idx serious error

type TableInserterQueryPerformerTestIdxSeriousError struct {
	TotalIdxHits           int
	SimulatedIdxErrorCount int
}

func (qp *TableInserterQueryPerformerTestIdxSeriousError) PerformInsertDataRecordWithRowid(gocqlSession gocqlshims.Session, pq *cql.PreparedQuery, preparedDataQueryParams []any, _ int) (map[string]any, bool, error) {
	return db.HelperPerformInsertDataRecordWithRowid(gocqlSession, pq, preparedDataQueryParams)
}

func (qp *TableInserterQueryPerformerTestIdxSeriousError) PerformInsertIdxRecordWithRowid(idxUniqueness sc.IdxUniqueness, gocqlSession gocqlshims.Session, pq *cql.PreparedQuery, preparedIdxQueryParams []any, retryCount int) (map[string]any, int, bool, error) {
	qp.TotalIdxHits++
	if retryCount > 0 || qp.SimulatedIdxErrorCount >= 1 {
		return db.HelperPerformInsertIdxRecordWithRowid(idxUniqueness, gocqlSession, pq, preparedIdxQueryParams, retryCount)
	}

	qp.SimulatedIdxErrorCount++
	// log: some serious error; cannot write to idx table
	// give up immediately and report failure
	return nil, retryCount, false, errors.New("test scenario: idx table " + cql.ErrorSomeSeriousError)
}

func TestIdxSeriousErrorLookup(t *testing.T) {
	os.Remove("/tmp/capi_out/lookup_quicktest/order_date_value_grouped_inner.csv")
	os.Remove("/tmp/capi_out/lookup_quicktest/order_date_value_grouped_left_outer.csv")
	os.Remove("/tmp/capi_out/lookup_quicktest/order_item_date_inner.csv")
	os.Remove("/tmp/capi_out/lookup_quicktest/order_item_date_left_outer.csv")

	ks := "ks_idx_serious_error_lookup"

	envConfig := env.EnvConfig{
		Cassandra:                         env.CassandraConfig{WriterWorkers: 1},
		Log:                               env.LogConfig{Level: "INFO"},
		CustomProcessorDefFactoryInstance: &TestProcessorDefFactory{},
		CustomProcessorsSettings:          getTestProcessorSettings(),
		UseGocqlmem:                       true,
	}
	sc.ScriptDefCache = sc.NewScriptDefCache()
	NodeDependencyReadynessCache = NewNodeDependencyReadynessCache()

	logger, err := l.NewLoggerFromEnvConfig(&envConfig, "unittest")
	assert.Nil(t, err)
	logger.PushF("TestIdxSeriousErrorLookup")
	defer logger.PopF()

	mqProducer := mq.TestInmemProducer{}

	gocqlmemSession, cassandraEngineType, err := db.NewSession(&envConfig, ks, db.CreateKeyspaceOnConnect)
	assert.Nil(t, err)

	_, err = StartRun(&envConfig, logger, &mqProducer, "/tmp/capi_cfg/lookup_quicktest/script_quick.yaml", "/tmp/capi_cfg/lookup_quicktest/script_params_quick_fs_one.yaml", gocqlmemSession, cassandraEngineType, ks, []string{"read_orders", "read_order_items"}, "test run")
	assert.Nil(t, err)

	var runStatus wfmodel.RunStatusType

	// Verify run status
	runHistory, err := GetRunHistory(gocqlmemSession, ks)
	assert.Nil(t, err)
	runStatus = runHistory[len(runHistory)-1].Status
	assert.Equal(t, wfmodel.RunStart, runStatus)

	for {
		msg := mqProducer.PeekHead()
		if msg == nil {
			break
		}
		queryPerformer := TableInserterQueryPerformerTestIdxSeriousError{}
		ackCmd := ProcessDataBatchMsg(&envConfig, logger, msg, 0, nil, ctx.TableInserterProperties{
			QueryPerformer:               &queryPerformer,
			DoesNotExistPauseMillis:      10, // speed it up for testing
			OperationTimedOutPauseMillis: 10, // speed it up for testing
		})
		if ackCmd == mq.AcknowledgerCmdAck {
			mqProducer.RemoveHead()
		} else {
			mqProducer.MoveHeadToTail()
		}
	}

	// Verify run status
	runHistory, err = GetRunHistory(gocqlmemSession, ks)
	assert.Nil(t, err)
	runStatus = runHistory[len(runHistory)-1].Status
	assert.Equal(t, wfmodel.RunComplete, runStatus)

	// Verify node statuses
	nodeHistory, err := GetNodeHistoryForRuns(gocqlmemSession, ks, []int16{int16(1)})
	assert.Nil(t, err)
	newNodeRunStatusMap := map[string]wfmodel.NodeBatchStatusType{}
	for _, nodeEvent := range nodeHistory {
		newNodeRunStatusMap[nodeEvent.ScriptNode] = nodeEvent.Status
	}

	// For each node, verify batch statuses
	for nodeName, nodeStatus := range newNodeRunStatusMap {
		assert.Equal(t, wfmodel.NodeBatchFail, nodeStatus, fmt.Sprintf("node %s supposed to fail", nodeName))
		// Make sure all batches for this node started then failed
		batchEvents, err := GetBatchHistoryForRunAndNode(gocqlmemSession, ks, int16(1), nodeName)
		assert.Nil(t, err)
		if nodeName == "read_orders" || nodeName == "read_order_items" {
			// These nodes start and fail
			assert.Equal(t, 2, len(batchEvents), nodeName)
			assert.Equal(t, wfmodel.NodeBatchStart, batchEvents[0].Status, nodeName)
			assert.Equal(t, wfmodel.NodeBatchFail, batchEvents[1].Status, nodeName)
			assert.True(t, strings.Contains(batchEvents[1].Comment, cql.ErrorSomeSeriousError))
		} else {
			// These nodes failed without starting
			for _, event := range batchEvents {
				assert.Equal(t, wfmodel.NodeBatchFail, event.Status, nodeName)
				assert.True(t, strings.Contains(event.Comment, "some dependency nodes"))
			}
		}
	}
	assert.Nil(t, gocqlmemSession.Query(fmt.Sprintf("DROP keyspace %s;", ks)).Exec())
	gocqlmemSession.Close()
}

func TestIdxSeriousErrorFannieMae(t *testing.T) {
	os.Remove("/tmp/capi_out/fannie_mae_apitest/deal_seller_summaries.csv")
	os.Remove("/tmp/capi_out/fannie_mae_apitest/deal_summaries.csv")
	os.Remove("/tmp/capi_out/fannie_mae_apitest/loan_smrs_clcltd.csv")

	ks := "ks_idx_serious_error_fannie_mae"

	envConfig := env.EnvConfig{
		Cassandra:                         env.CassandraConfig{WriterWorkers: 1},
		Log:                               env.LogConfig{Level: "INFO"},
		CustomProcessorDefFactoryInstance: &TestProcessorDefFactory{},
		CustomProcessorsSettings:          getTestProcessorSettings(),
		UseGocqlmem:                       true,
	}
	sc.ScriptDefCache = sc.NewScriptDefCache()
	NodeDependencyReadynessCache = NewNodeDependencyReadynessCache()

	logger, err := l.NewLoggerFromEnvConfig(&envConfig, "unittest")
	assert.Nil(t, err)
	logger.PushF("TestIdxSeriousErrorFannieMae")
	defer logger.PopF()

	mqProducer := mq.TestInmemProducer{}

	gocqlmemSession, cassandraEngineType, err := db.NewSession(&envConfig, ks, db.CreateKeyspaceOnConnect)
	assert.Nil(t, err)

	_, err = StartRun(&envConfig, logger, &mqProducer, "/tmp/capi_cfg/fannie_mae_apitest/script_api.json", "/tmp/capi_cfg/fannie_mae_apitest/script_params_api.json", gocqlmemSession, cassandraEngineType, ks, []string{"01_read_payments"}, "test run")
	assert.Nil(t, err)

	var runStatus wfmodel.RunStatusType

	// Verify run status
	runHistory, err := GetRunHistory(gocqlmemSession, ks)
	assert.Nil(t, err)
	runStatus = runHistory[len(runHistory)-1].Status
	assert.Equal(t, wfmodel.RunStart, runStatus)

	for {
		msg := mqProducer.PeekHead()
		if msg == nil {
			break
		}
		queryPerformer := TableInserterQueryPerformerTestIdxSeriousError{}
		ackCmd := ProcessDataBatchMsg(&envConfig, logger, msg, 0, nil, ctx.TableInserterProperties{
			QueryPerformer:               &queryPerformer,
			DoesNotExistPauseMillis:      10, // speed it up for testing
			OperationTimedOutPauseMillis: 10, // speed it up for testing
		})
		if ackCmd == mq.AcknowledgerCmdAck {
			mqProducer.RemoveHead()
		} else {
			mqProducer.MoveHeadToTail()
		}
	}

	// Verify run status
	runHistory, err = GetRunHistory(gocqlmemSession, ks)
	assert.Nil(t, err)
	runStatus = runHistory[len(runHistory)-1].Status
	assert.Equal(t, wfmodel.RunComplete, runStatus)

	// Verify node statuses
	nodeHistory, err := GetNodeHistoryForRuns(gocqlmemSession, ks, []int16{int16(1)})
	assert.Nil(t, err)
	newNodeRunStatusMap := map[string]wfmodel.NodeBatchStatusType{}
	for _, nodeEvent := range nodeHistory {
		newNodeRunStatusMap[nodeEvent.ScriptNode] = nodeEvent.Status
	}

	// For each node, verify batch statuses
	for nodeName, nodeStatus := range newNodeRunStatusMap {
		assert.Equal(t, wfmodel.NodeBatchFail, nodeStatus, fmt.Sprintf("node %s supposed to fail", nodeName))
		// Make sure all batches for this node started then failed
		batchEvents, err := GetBatchHistoryForRunAndNode(gocqlmemSession, ks, int16(1), nodeName)
		assert.Nil(t, err)
		if nodeName == "01_read_payments" {
			// These nodes start and fail
			assert.Equal(t, 2, len(batchEvents), nodeName)
			assert.Equal(t, wfmodel.NodeBatchStart, batchEvents[0].Status, nodeName)
			assert.Equal(t, wfmodel.NodeBatchFail, batchEvents[1].Status, nodeName)
			assert.True(t, strings.Contains(batchEvents[1].Comment, cql.ErrorSomeSeriousError))
		} else {
			// These nodes failed without starting
			for _, event := range batchEvents {
				assert.Equal(t, wfmodel.NodeBatchFail, event.Status, nodeName)
				assert.True(t, strings.Contains(event.Comment, "some dependency nodes"))
			}
		}
	}
	assert.Nil(t, gocqlmemSession.Query(fmt.Sprintf("DROP keyspace %s;", ks)).Exec())
	gocqlmemSession.Close()
}

// Data not applied

type TableInserterQueryPerformerTestDataNotApplied struct {
	TotalDataHits           int
	SimulatedDataErrorCount int
}

func (qp *TableInserterQueryPerformerTestDataNotApplied) PerformInsertDataRecordWithRowid(gocqlSession gocqlshims.Session, pq *cql.PreparedQuery, preparedDataQueryParams []any, retryCount int) (map[string]any, bool, error) {
	qp.TotalDataHits++
	if retryCount > 0 || qp.SimulatedDataErrorCount >= 1 {
		return db.HelperPerformInsertDataRecordWithRowid(gocqlSession, pq, preparedDataQueryParams)
	}

	qp.SimulatedDataErrorCount++
	// log warning: duplicate rowid not written [INSERT INTO ...], existing record [...], table retry count 0
	// This will trigger non-fatal ErrDuplicateRowid error
	// 	retry with new rowid and succeed
	// 	isApplied = false
	return nil, false, nil
}

func (qp *TableInserterQueryPerformerTestDataNotApplied) PerformInsertIdxRecordWithRowid(idxUniqueness sc.IdxUniqueness, gocqlSession gocqlshims.Session, pq *cql.PreparedQuery, preparedIdxQueryParams []any, retryCount int) (map[string]any, int, bool, error) {
	return db.HelperPerformInsertIdxRecordWithRowid(idxUniqueness, gocqlSession, pq, preparedIdxQueryParams, retryCount)
}

func TestDataNotAppliedLookup(t *testing.T) {
	os.Remove("/tmp/capi_out/lookup_quicktest/order_date_value_grouped_inner.csv")
	os.Remove("/tmp/capi_out/lookup_quicktest/order_date_value_grouped_left_outer.csv")
	os.Remove("/tmp/capi_out/lookup_quicktest/order_item_date_inner.csv")
	os.Remove("/tmp/capi_out/lookup_quicktest/order_item_date_left_outer.csv")

	ks := "ks_data_not_applied_lookup"

	envConfig := env.EnvConfig{
		Cassandra:                         env.CassandraConfig{WriterWorkers: 2},
		Log:                               env.LogConfig{Level: "INFO"},
		CustomProcessorDefFactoryInstance: &TestProcessorDefFactory{},
		CustomProcessorsSettings:          getTestProcessorSettings(),
		UseGocqlmem:                       true,
	}
	sc.ScriptDefCache = sc.NewScriptDefCache()
	NodeDependencyReadynessCache = NewNodeDependencyReadynessCache()

	logger, err := l.NewLoggerFromEnvConfig(&envConfig, "unittest")
	assert.Nil(t, err)
	logger.PushF("TestDataNotAppliedLookup")
	defer logger.PopF()

	mqProducer := mq.TestInmemProducer{}

	gocqlmemSession, cassandraEngineType, err := db.NewSession(&envConfig, ks, db.CreateKeyspaceOnConnect)
	assert.Nil(t, err)

	_, err = StartRun(&envConfig, logger, &mqProducer, "/tmp/capi_cfg/lookup_quicktest/script_quick.yaml", "/tmp/capi_cfg/lookup_quicktest/script_params_quick_fs_one.yaml", gocqlmemSession, cassandraEngineType, ks, []string{"read_orders", "read_order_items"}, "test run")
	assert.Nil(t, err)

	var runStatus wfmodel.RunStatusType

	// Verify run status
	runHistory, err := GetRunHistory(gocqlmemSession, ks)
	assert.Nil(t, err)
	runStatus = runHistory[len(runHistory)-1].Status
	assert.Equal(t, wfmodel.RunStart, runStatus)

	for {
		msg := mqProducer.PeekHead()
		if msg == nil {
			break
		}
		queryPerformer := TableInserterQueryPerformerTestDataNotApplied{}
		ackCmd := ProcessDataBatchMsg(&envConfig, logger, msg, 0, nil, ctx.TableInserterProperties{
			QueryPerformer:               &queryPerformer,
			DoesNotExistPauseMillis:      10, // speed it up for testing
			OperationTimedOutPauseMillis: 10, // speed it up for testing
		})
		if ackCmd == mq.AcknowledgerCmdAck {
			mqProducer.RemoveHead()
		} else {
			mqProducer.MoveHeadToTail()
		}

		// Make sure that fake errors were actually introduced
		if queryPerformer.TotalDataHits > 0 {
			if strings.HasPrefix(msg.TargetNodeName, "file_") {
				assert.Equal(t, 0, queryPerformer.SimulatedDataErrorCount, msg.TargetNodeName)
			} else {
				assert.Equal(t, 1, queryPerformer.SimulatedDataErrorCount, msg.TargetNodeName)
			}
		}
	}

	// Verify run status
	runHistory, err = GetRunHistory(gocqlmemSession, ks)
	assert.Nil(t, err)
	runStatus = runHistory[len(runHistory)-1].Status
	assert.Equal(t, wfmodel.RunComplete, runStatus)

	err = compareCsvs("/tmp/capi_out/lookup_quicktest/order_date_value_grouped_inner_baseline.csv", "/tmp/capi_out/lookup_quicktest/order_date_value_grouped_inner.csv")
	assert.Nil(t, err)
	err = compareCsvs("/tmp/capi_out/lookup_quicktest/order_date_value_grouped_left_outer_baseline.csv", "/tmp/capi_out/lookup_quicktest/order_date_value_grouped_left_outer.csv")
	assert.Nil(t, err)
	err = compareCsvs("/tmp/capi_out/lookup_quicktest/order_item_date_inner_baseline.csv", "/tmp/capi_out/lookup_quicktest/order_item_date_inner.csv")
	assert.Nil(t, err)
	err = compareCsvs("/tmp/capi_out/lookup_quicktest/order_item_date_left_outer_baseline.csv", "/tmp/capi_out/lookup_quicktest/order_item_date_left_outer.csv")
	assert.Nil(t, err)

	assert.Nil(t, gocqlmemSession.Query(fmt.Sprintf("DROP keyspace %s;", ks)).Exec())
	gocqlmemSession.Close()
}

func TestDataNotAppliedFannieMae(t *testing.T) {
	os.Remove("/tmp/capi_out/fannie_mae_apitest/deal_seller_summaries.csv")
	os.Remove("/tmp/capi_out/fannie_mae_apitest/deal_summaries.csv")
	os.Remove("/tmp/capi_out/fannie_mae_apitest/loan_smrs_clcltd.csv")

	ks := "ks_data_not_applied_fannie_mae"

	envConfig := env.EnvConfig{
		Cassandra:                         env.CassandraConfig{WriterWorkers: 2},
		Log:                               env.LogConfig{Level: "INFO"},
		CustomProcessorDefFactoryInstance: &TestProcessorDefFactory{},
		CustomProcessorsSettings:          getTestProcessorSettings(),
		UseGocqlmem:                       true,
	}
	sc.ScriptDefCache = sc.NewScriptDefCache()
	NodeDependencyReadynessCache = NewNodeDependencyReadynessCache()

	logger, err := l.NewLoggerFromEnvConfig(&envConfig, "unittest")
	assert.Nil(t, err)
	logger.PushF("TestDataNotAppliedFannieMae")
	defer logger.PopF()

	mqProducer := mq.TestInmemProducer{}

	gocqlmemSession, cassandraEngineType, err := db.NewSession(&envConfig, ks, db.CreateKeyspaceOnConnect)
	assert.Nil(t, err)

	_, err = StartRun(&envConfig, logger, &mqProducer, "/tmp/capi_cfg/fannie_mae_apitest/script_api.json", "/tmp/capi_cfg/fannie_mae_apitest/script_params_api.json", gocqlmemSession, cassandraEngineType, ks, []string{"01_read_payments"}, "test run")
	assert.Nil(t, err)

	var runStatus wfmodel.RunStatusType

	// Verify run status
	runHistory, err := GetRunHistory(gocqlmemSession, ks)
	assert.Nil(t, err)
	runStatus = runHistory[len(runHistory)-1].Status
	assert.Equal(t, wfmodel.RunStart, runStatus)

	for {
		msg := mqProducer.PeekHead()
		if msg == nil {
			break
		}
		queryPerformer := TableInserterQueryPerformerTestDataNotApplied{}
		ackCmd := ProcessDataBatchMsg(&envConfig, logger, msg, 0, nil, ctx.TableInserterProperties{
			QueryPerformer:               &queryPerformer,
			DoesNotExistPauseMillis:      10,  // speed it up for testing
			OperationTimedOutPauseMillis: 100, // speed it up for testing
		})
		if ackCmd == mq.AcknowledgerCmdAck {
			mqProducer.RemoveHead()
		} else {
			mqProducer.MoveHeadToTail()
		}

		// Make sure that fake errors were actually introduced
		if queryPerformer.TotalDataHits > 0 {
			if strings.Contains(msg.TargetNodeName, "write_file") {
				assert.Equal(t, 0, queryPerformer.SimulatedDataErrorCount, msg.TargetNodeName)
			} else {
				assert.Equal(t, 1, queryPerformer.SimulatedDataErrorCount, msg.TargetNodeName)
			}
		}
	}

	// Verify run status
	runHistory, err = GetRunHistory(gocqlmemSession, ks)
	assert.Nil(t, err)
	runStatus = runHistory[len(runHistory)-1].Status
	assert.Equal(t, wfmodel.RunComplete, runStatus)

	err = compareCsvs("/tmp/capi_out/fannie_mae_apitest/deal_seller_summaries_baseline.csv", "/tmp/capi_out/fannie_mae_apitest/deal_seller_summaries.csv")
	assert.Nil(t, err)
	err = compareCsvs("/tmp/capi_out/fannie_mae_apitest/deal_summaries_baseline.csv", "/tmp/capi_out/fannie_mae_apitest/deal_summaries.csv")
	assert.Nil(t, err)
	err = compareCsvs("/tmp/capi_out/fannie_mae_apitest/loan_smrs_clcltd_baseline.csv", "/tmp/capi_out/fannie_mae_apitest/loan_smrs_clcltd.csv")
	assert.Nil(t, err)

	assert.Nil(t, gocqlmemSession.Query(fmt.Sprintf("DROP keyspace %s;", ks)).Exec())
	gocqlmemSession.Close()
}

// IdxNotAppliedSamePresentFirstRetry

type TableInserterQueryPerformerTestIdxNotAppliedSamePresentFirstRetry struct {
	TotalIdxHits           int
	SimulatedIdxErrorCount int
}

func (qp *TableInserterQueryPerformerTestIdxNotAppliedSamePresentFirstRetry) PerformInsertDataRecordWithRowid(gocqlSession gocqlshims.Session, pq *cql.PreparedQuery, preparedDataQueryParams []any, _ int) (map[string]any, bool, error) {
	return db.HelperPerformInsertDataRecordWithRowid(gocqlSession, pq, preparedDataQueryParams)
}

func (qp *TableInserterQueryPerformerTestIdxNotAppliedSamePresentFirstRetry) PerformInsertIdxRecordWithRowid(idxUniqueness sc.IdxUniqueness, gocqlSession gocqlshims.Session, pq *cql.PreparedQuery, preparedIdxQueryParams []any, retryCount int) (map[string]any, int, bool, error) {
	qp.TotalIdxHits++
	if retryCount > 0 || qp.SimulatedIdxErrorCount >= 1 {
		return db.HelperPerformInsertIdxRecordWithRowid(idxUniqueness, gocqlSession, pq, preparedIdxQueryParams, retryCount)
	}

	qp.SimulatedIdxErrorCount++
	// log: cannot write duplicate index key [%s] and proper rowid with %s,%d on retry 0, existing record [%v], assuming it was some other writer, throwing error %w
	// give up immediately and report failure
	existingIdxRow := map[string]any{}
	existingIdxRow["key"] = pq.Qb.PreparedColumnData.Values[pq.Qb.PreparedColumnData.ColumnIdxMap["key"]]
	existingIdxRow["rowid"] = pq.Qb.PreparedColumnData.Values[pq.Qb.PreparedColumnData.ColumnIdxMap["rowid"]]
	return existingIdxRow, retryCount, false, nil
}

func TestIdxNotAppliedSamePresentFirstRetryLookup(t *testing.T) {
	os.Remove("/tmp/capi_out/lookup_quicktest/order_date_value_grouped_inner.csv")
	os.Remove("/tmp/capi_out/lookup_quicktest/order_date_value_grouped_left_outer.csv")
	os.Remove("/tmp/capi_out/lookup_quicktest/order_item_date_inner.csv")
	os.Remove("/tmp/capi_out/lookup_quicktest/order_item_date_left_outer.csv")

	ks := "ks_idx_not_applied_same_present_first_retry_lookup"

	envConfig := env.EnvConfig{
		Cassandra:                         env.CassandraConfig{WriterWorkers: 1},
		Log:                               env.LogConfig{Level: "INFO"},
		CustomProcessorDefFactoryInstance: &TestProcessorDefFactory{},
		CustomProcessorsSettings:          getTestProcessorSettings(),
		UseGocqlmem:                       true,
	}
	sc.ScriptDefCache = sc.NewScriptDefCache()
	NodeDependencyReadynessCache = NewNodeDependencyReadynessCache()

	logger, err := l.NewLoggerFromEnvConfig(&envConfig, "unittest")
	assert.Nil(t, err)
	logger.PushF("TestIdxSeriousErrorLookup")
	defer logger.PopF()

	mqProducer := mq.TestInmemProducer{}

	gocqlmemSession, cassandraEngineType, err := db.NewSession(&envConfig, ks, db.CreateKeyspaceOnConnect)
	assert.Nil(t, err)

	_, err = StartRun(&envConfig, logger, &mqProducer, "/tmp/capi_cfg/lookup_quicktest/script_quick.yaml", "/tmp/capi_cfg/lookup_quicktest/script_params_quick_fs_one.yaml", gocqlmemSession, cassandraEngineType, ks, []string{"read_orders", "read_order_items"}, "test run")
	assert.Nil(t, err)

	var runStatus wfmodel.RunStatusType

	// Verify run status
	runHistory, err := GetRunHistory(gocqlmemSession, ks)
	assert.Nil(t, err)
	runStatus = runHistory[len(runHistory)-1].Status
	assert.Equal(t, wfmodel.RunStart, runStatus)

	for {
		msg := mqProducer.PeekHead()
		if msg == nil {
			break
		}
		queryPerformer := TableInserterQueryPerformerTestIdxNotAppliedSamePresentFirstRetry{}
		ackCmd := ProcessDataBatchMsg(&envConfig, logger, msg, 0, nil, ctx.TableInserterProperties{
			QueryPerformer:               &queryPerformer,
			DoesNotExistPauseMillis:      10, // speed it up for testing
			OperationTimedOutPauseMillis: 10, // speed it up for testing
		})
		if ackCmd == mq.AcknowledgerCmdAck {
			mqProducer.RemoveHead()
		} else {
			mqProducer.MoveHeadToTail()
		}
	}

	// Verify run status
	runHistory, err = GetRunHistory(gocqlmemSession, ks)
	assert.Nil(t, err)
	runStatus = runHistory[len(runHistory)-1].Status
	assert.Equal(t, wfmodel.RunComplete, runStatus)

	// Verify node statuses
	nodeHistory, err := GetNodeHistoryForRuns(gocqlmemSession, ks, []int16{int16(1)})
	assert.Nil(t, err)
	newNodeRunStatusMap := map[string]wfmodel.NodeBatchStatusType{}
	for _, nodeEvent := range nodeHistory {
		newNodeRunStatusMap[nodeEvent.ScriptNode] = nodeEvent.Status
	}

	// For each node, verify batch statuses
	for nodeName, nodeStatus := range newNodeRunStatusMap {
		assert.Equal(t, wfmodel.NodeBatchFail, nodeStatus, fmt.Sprintf("node %s supposed to fail", nodeName))
		// Make sure all batches for this node started then failed
		batchEvents, err := GetBatchHistoryForRunAndNode(gocqlmemSession, ks, int16(1), nodeName)
		assert.Nil(t, err)
		if nodeName == "read_orders" || nodeName == "read_order_items" {
			// These nodes start and fail
			assert.Equal(t, 2, len(batchEvents), nodeName)
			assert.Equal(t, wfmodel.NodeBatchStart, batchEvents[0].Status, nodeName)
			assert.Equal(t, wfmodel.NodeBatchFail, batchEvents[1].Status, nodeName)
			assert.True(t, strings.Contains(batchEvents[1].Comment, "assuming it was some other writer, throwing error duplicate key"))
		} else {
			// These nodes failed without starting
			for _, event := range batchEvents {
				assert.Equal(t, wfmodel.NodeBatchFail, event.Status, nodeName)
				assert.True(t, strings.Contains(event.Comment, "some dependency nodes"))
			}
		}
	}
	assert.Nil(t, gocqlmemSession.Query(fmt.Sprintf("DROP keyspace %s;", ks)).Exec())
	gocqlmemSession.Close()
}

func TestIdxNotAppliedSamePresentFirstRetryFannieMae(t *testing.T) {
	os.Remove("/tmp/capi_out/fannie_mae_apitest/deal_seller_summaries.csv")
	os.Remove("/tmp/capi_out/fannie_mae_apitest/deal_summaries.csv")
	os.Remove("/tmp/capi_out/fannie_mae_apitest/loan_smrs_clcltd.csv")

	ks := "ks_idx_not_applied_same_present_first_retry_fannie_mae"

	envConfig := env.EnvConfig{
		Cassandra:                         env.CassandraConfig{WriterWorkers: 1},
		Log:                               env.LogConfig{Level: "INFO"},
		CustomProcessorDefFactoryInstance: &TestProcessorDefFactory{},
		CustomProcessorsSettings:          getTestProcessorSettings(),
		UseGocqlmem:                       true,
	}
	sc.ScriptDefCache = sc.NewScriptDefCache()
	NodeDependencyReadynessCache = NewNodeDependencyReadynessCache()

	logger, err := l.NewLoggerFromEnvConfig(&envConfig, "unittest")
	assert.Nil(t, err)
	logger.PushF("TestIdxSeriousErrorFannieMae")
	defer logger.PopF()

	mqProducer := mq.TestInmemProducer{}

	gocqlmemSession, cassandraEngineType, err := db.NewSession(&envConfig, ks, db.CreateKeyspaceOnConnect)
	assert.Nil(t, err)

	_, err = StartRun(&envConfig, logger, &mqProducer, "/tmp/capi_cfg/fannie_mae_apitest/script_api.json", "/tmp/capi_cfg/fannie_mae_apitest/script_params_api.json", gocqlmemSession, cassandraEngineType, ks, []string{"01_read_payments"}, "test run")
	assert.Nil(t, err)

	var runStatus wfmodel.RunStatusType

	// Verify run status
	runHistory, err := GetRunHistory(gocqlmemSession, ks)
	assert.Nil(t, err)
	runStatus = runHistory[len(runHistory)-1].Status
	assert.Equal(t, wfmodel.RunStart, runStatus)

	for {
		msg := mqProducer.PeekHead()
		if msg == nil {
			break
		}
		queryPerformer := TableInserterQueryPerformerTestIdxNotAppliedSamePresentFirstRetry{}
		ackCmd := ProcessDataBatchMsg(&envConfig, logger, msg, 0, nil, ctx.TableInserterProperties{
			QueryPerformer:               &queryPerformer,
			DoesNotExistPauseMillis:      10,  // speed it up for testing
			OperationTimedOutPauseMillis: 100, // speed it up for testing
		})
		if ackCmd == mq.AcknowledgerCmdAck {
			mqProducer.RemoveHead()
		} else {
			mqProducer.MoveHeadToTail()
		}
	}

	// Verify run status
	runHistory, err = GetRunHistory(gocqlmemSession, ks)
	assert.Nil(t, err)
	runStatus = runHistory[len(runHistory)-1].Status
	assert.Equal(t, wfmodel.RunComplete, runStatus)

	// Verify node statuses
	nodeHistory, err := GetNodeHistoryForRuns(gocqlmemSession, ks, []int16{int16(1)})
	assert.Nil(t, err)
	newNodeRunStatusMap := map[string]wfmodel.NodeBatchStatusType{}
	for _, nodeEvent := range nodeHistory {
		newNodeRunStatusMap[nodeEvent.ScriptNode] = nodeEvent.Status
	}

	// For each node, verify batch statuses
	for nodeName, nodeStatus := range newNodeRunStatusMap {
		assert.Equal(t, wfmodel.NodeBatchFail, nodeStatus, fmt.Sprintf("node %s supposed to fail", nodeName))
		// Make sure all batches for this node started then failed
		batchEvents, err := GetBatchHistoryForRunAndNode(gocqlmemSession, ks, int16(1), nodeName)
		assert.Nil(t, err)
		if nodeName == "01_read_payments" {
			// These nodes start and fail
			assert.Equal(t, 2, len(batchEvents), nodeName)
			assert.Equal(t, wfmodel.NodeBatchStart, batchEvents[0].Status, nodeName)
			assert.Equal(t, wfmodel.NodeBatchFail, batchEvents[1].Status, nodeName)
			assert.True(t, strings.Contains(batchEvents[1].Comment, "assuming it was some other writer, throwing error duplicate key"))
		} else {
			// These nodes failed without starting
			for _, event := range batchEvents {
				assert.Equal(t, wfmodel.NodeBatchFail, event.Status, nodeName)
				assert.True(t, strings.Contains(event.Comment, "some dependency nodes"))
			}
		}
	}
	assert.Nil(t, gocqlmemSession.Query(fmt.Sprintf("DROP keyspace %s;", ks)).Exec())
	gocqlmemSession.Close()
}

// IdxNotAppliedSamePresentSecondRetry

type TableInserterQueryPerformerTestIdxNotAppliedSamePresentSecondRetry struct {
	TotalIdxHits           int
	SimulatedIdxErrorCount int
}

func (qp *TableInserterQueryPerformerTestIdxNotAppliedSamePresentSecondRetry) PerformInsertDataRecordWithRowid(gocqlSession gocqlshims.Session, pq *cql.PreparedQuery, preparedDataQueryParams []any, _ int) (map[string]any, bool, error) {
	return db.HelperPerformInsertDataRecordWithRowid(gocqlSession, pq, preparedDataQueryParams)
}

func (qp *TableInserterQueryPerformerTestIdxNotAppliedSamePresentSecondRetry) PerformInsertIdxRecordWithRowid(idxUniqueness sc.IdxUniqueness, gocqlSession gocqlshims.Session, pq *cql.PreparedQuery, preparedIdxQueryParams []any, retryCount int) (map[string]any, int, bool, error) {
	qp.TotalIdxHits++

	if retryCount > 0 || qp.SimulatedIdxErrorCount >= 1 {
		return db.HelperPerformInsertIdxRecordWithRowid(idxUniqueness, gocqlSession, pq, preparedIdxQueryParams, retryCount)
	}

	qp.SimulatedIdxErrorCount++
	if idxUniqueness == sc.IdxNonUnique {
		existingIdxRow, _, _, err := db.HelperPerformInsertIdxRecordWithRowid(idxUniqueness, gocqlSession, pq, preparedIdxQueryParams, retryCount)
		if err != nil {
			return existingIdxRow, retryCount, false, fmt.Errorf("PerformInsertIdxRecordWithRowid got %s", err.Error())
		}
		// log: cannot write duplicate index key [%s] and proper rowid with %s,%d on retry 0, existing record [%v], assuming it was some other writer, throwing error %w
		// give up immediately and report failure
		existingIdxRow["key"] = pq.Qb.PreparedColumnData.Values[pq.Qb.PreparedColumnData.ColumnIdxMap["key"]]
		existingIdxRow["rowid"] = pq.Qb.PreparedColumnData.Values[pq.Qb.PreparedColumnData.ColumnIdxMap["rowid"]]
		return existingIdxRow, retryCount + 1, false, nil
	}

	// TODO: to test insertDistinctIdxAndDataRecords's "found orphan distinct idx record for key", we need to call
	// db.HelperPerformInsertIdxRecordWithRowid(idxUniqueness, gocqlSession, pq, preparedIdxQueryParams, retryCount)
	// and if it is not applied (because it's DISTINCT idx), delete the data (not idx) record with this rowid, and
	// return existingIdxRow, retryCount, false, nil
	// But it's too complex for now, leave till we really want to have 100% coverage of insertDistinctIdxAndDataRecords.
	return db.HelperPerformInsertIdxRecordWithRowid(idxUniqueness, gocqlSession, pq, preparedIdxQueryParams, retryCount)
}

func TestIdxNotAppliedSamePresentSecondRetryLookup(t *testing.T) {
	os.Remove("/tmp/capi_out/lookup_quicktest/order_date_value_grouped_inner.csv")
	os.Remove("/tmp/capi_out/lookup_quicktest/order_date_value_grouped_left_outer.csv")
	os.Remove("/tmp/capi_out/lookup_quicktest/order_item_date_inner.csv")
	os.Remove("/tmp/capi_out/lookup_quicktest/order_item_date_left_outer.csv")

	ks := "ks_idx_not_applied_same_present_second_retry_lookup"

	envConfig := env.EnvConfig{
		Cassandra:                         env.CassandraConfig{WriterWorkers: 2},
		Log:                               env.LogConfig{Level: "INFO"},
		CustomProcessorDefFactoryInstance: &TestProcessorDefFactory{},
		CustomProcessorsSettings:          getTestProcessorSettings(),
		UseGocqlmem:                       true,
	}
	sc.ScriptDefCache = sc.NewScriptDefCache()
	NodeDependencyReadynessCache = NewNodeDependencyReadynessCache()

	logger, err := l.NewLoggerFromEnvConfig(&envConfig, "unittest")
	assert.Nil(t, err)
	logger.PushF("TestIdxSeriousErrorLookup")
	defer logger.PopF()

	mqProducer := mq.TestInmemProducer{}

	gocqlmemSession, cassandraEngineType, err := db.NewSession(&envConfig, ks, db.CreateKeyspaceOnConnect)
	assert.Nil(t, err)

	_, err = StartRun(&envConfig, logger, &mqProducer, "/tmp/capi_cfg/lookup_quicktest/script_quick.yaml", "/tmp/capi_cfg/lookup_quicktest/script_params_quick_fs_one.yaml", gocqlmemSession, cassandraEngineType, ks, []string{"read_orders", "read_order_items"}, "test run")
	assert.Nil(t, err)

	var runStatus wfmodel.RunStatusType

	// Verify run status
	runHistory, err := GetRunHistory(gocqlmemSession, ks)
	assert.Nil(t, err)
	runStatus = runHistory[len(runHistory)-1].Status
	assert.Equal(t, wfmodel.RunStart, runStatus)

	for {
		msg := mqProducer.PeekHead()
		if msg == nil {
			break
		}
		queryPerformer := TableInserterQueryPerformerTestIdxNotAppliedSamePresentSecondRetry{}
		ackCmd := ProcessDataBatchMsg(&envConfig, logger, msg, 0, nil, ctx.TableInserterProperties{
			QueryPerformer:               &queryPerformer,
			DoesNotExistPauseMillis:      10, // speed it up for testing
			OperationTimedOutPauseMillis: 10, // speed it up for testing
		})
		if ackCmd == mq.AcknowledgerCmdAck {
			mqProducer.RemoveHead()
		} else {
			mqProducer.MoveHeadToTail()
		}
	}

	// Verify run status
	runHistory, err = GetRunHistory(gocqlmemSession, ks)
	assert.Nil(t, err)
	runStatus = runHistory[len(runHistory)-1].Status
	assert.Equal(t, wfmodel.RunComplete, runStatus)

	err = compareCsvs("/tmp/capi_out/lookup_quicktest/order_date_value_grouped_inner_baseline.csv", "/tmp/capi_out/lookup_quicktest/order_date_value_grouped_inner.csv")
	assert.Nil(t, err)
	err = compareCsvs("/tmp/capi_out/lookup_quicktest/order_date_value_grouped_left_outer_baseline.csv", "/tmp/capi_out/lookup_quicktest/order_date_value_grouped_left_outer.csv")
	assert.Nil(t, err)
	err = compareCsvs("/tmp/capi_out/lookup_quicktest/order_item_date_inner_baseline.csv", "/tmp/capi_out/lookup_quicktest/order_item_date_inner.csv")
	assert.Nil(t, err)
	err = compareCsvs("/tmp/capi_out/lookup_quicktest/order_item_date_left_outer_baseline.csv", "/tmp/capi_out/lookup_quicktest/order_item_date_left_outer.csv")
	assert.Nil(t, err)

	assert.Nil(t, gocqlmemSession.Query(fmt.Sprintf("DROP keyspace %s;", ks)).Exec())
	gocqlmemSession.Close()
}

func TestIdxNotAppliedSamePresentSecondRetryFannieMae(t *testing.T) {
	os.Remove("/tmp/capi_out/fannie_mae_apitest/deal_seller_summaries.csv")
	os.Remove("/tmp/capi_out/fannie_mae_apitest/deal_summaries.csv")
	os.Remove("/tmp/capi_out/fannie_mae_apitest/loan_smrs_clcltd.csv")

	ks := "ks_idx_not_applied_same_present_second_retry_fannie_mae"

	envConfig := env.EnvConfig{
		Cassandra:                         env.CassandraConfig{WriterWorkers: 1},
		Log:                               env.LogConfig{Level: "INFO"},
		CustomProcessorDefFactoryInstance: &TestProcessorDefFactory{},
		CustomProcessorsSettings:          getTestProcessorSettings(),
		UseGocqlmem:                       true,
	}
	sc.ScriptDefCache = sc.NewScriptDefCache()
	NodeDependencyReadynessCache = NewNodeDependencyReadynessCache()

	logger, err := l.NewLoggerFromEnvConfig(&envConfig, "unittest")
	assert.Nil(t, err)
	logger.PushF("TestIdxSeriousErrorFannieMae")
	defer logger.PopF()

	mqProducer := mq.TestInmemProducer{}

	gocqlmemSession, cassandraEngineType, err := db.NewSession(&envConfig, ks, db.CreateKeyspaceOnConnect)
	assert.Nil(t, err)

	_, err = StartRun(&envConfig, logger, &mqProducer, "/tmp/capi_cfg/fannie_mae_apitest/script_api.json", "/tmp/capi_cfg/fannie_mae_apitest/script_params_api.json", gocqlmemSession, cassandraEngineType, ks, []string{"01_read_payments"}, "test run")
	assert.Nil(t, err)

	var runStatus wfmodel.RunStatusType

	// Verify run status
	runHistory, err := GetRunHistory(gocqlmemSession, ks)
	assert.Nil(t, err)
	runStatus = runHistory[len(runHistory)-1].Status
	assert.Equal(t, wfmodel.RunStart, runStatus)

	for {
		msg := mqProducer.PeekHead()
		if msg == nil {
			break
		}
		queryPerformer := TableInserterQueryPerformerTestIdxNotAppliedSamePresentSecondRetry{}
		ackCmd := ProcessDataBatchMsg(&envConfig, logger, msg, 0, nil, ctx.TableInserterProperties{
			QueryPerformer:               &queryPerformer,
			DoesNotExistPauseMillis:      10,  // speed it up for testing
			OperationTimedOutPauseMillis: 100, // speed it up for testing
		})
		if ackCmd == mq.AcknowledgerCmdAck {
			mqProducer.RemoveHead()
		} else {
			mqProducer.MoveHeadToTail()
		}
	}

	// Verify run status
	runHistory, err = GetRunHistory(gocqlmemSession, ks)
	assert.Nil(t, err)
	runStatus = runHistory[len(runHistory)-1].Status
	assert.Equal(t, wfmodel.RunComplete, runStatus)

	err = compareCsvs("/tmp/capi_out/fannie_mae_apitest/deal_seller_summaries_baseline.csv", "/tmp/capi_out/fannie_mae_apitest/deal_seller_summaries.csv")
	assert.Nil(t, err)
	err = compareCsvs("/tmp/capi_out/fannie_mae_apitest/deal_summaries_baseline.csv", "/tmp/capi_out/fannie_mae_apitest/deal_summaries.csv")
	assert.Nil(t, err)
	err = compareCsvs("/tmp/capi_out/fannie_mae_apitest/loan_smrs_clcltd_baseline.csv", "/tmp/capi_out/fannie_mae_apitest/loan_smrs_clcltd.csv")
	assert.Nil(t, err)

	assert.Nil(t, gocqlmemSession.Query(fmt.Sprintf("DROP keyspace %s;", ks)).Exec())
	gocqlmemSession.Close()
}

// TestIdxNotAppliedDiffPresent

type TableInserterQueryPerformerTestIdxNotAppliedDiffPresent struct {
	TotalIdxHits           int
	SimulatedIdxErrorCount int
}

func (qp *TableInserterQueryPerformerTestIdxNotAppliedDiffPresent) PerformInsertDataRecordWithRowid(gocqlSession gocqlshims.Session, pq *cql.PreparedQuery, preparedDataQueryParams []any, _ int) (map[string]any, bool, error) {
	return db.HelperPerformInsertDataRecordWithRowid(gocqlSession, pq, preparedDataQueryParams)
}

func (qp *TableInserterQueryPerformerTestIdxNotAppliedDiffPresent) PerformInsertIdxRecordWithRowid(idxUniqueness sc.IdxUniqueness, gocqlSession gocqlshims.Session, pq *cql.PreparedQuery, preparedIdxQueryParams []any, retryCount int) (map[string]any, int, bool, error) {
	qp.TotalIdxHits++
	if retryCount > 0 || qp.SimulatedIdxErrorCount >= 1 {
		return db.HelperPerformInsertIdxRecordWithRowid(idxUniqueness, gocqlSession, pq, preparedIdxQueryParams, retryCount)
	}

	qp.SimulatedIdxErrorCount++
	// log: cannot write duplicate index key [%s] with %s,%d on retry %d, existing record [%v], rowid is different, throwing error %w
	// give up immediately and report failure
	existingIdxRow := map[string]any{}
	existingIdxRow["key"] = pq.Qb.PreparedColumnData.Values[pq.Qb.PreparedColumnData.ColumnIdxMap["key"]]
	existingIdxRow["rowid"] = -1 // Pray it's different than the passed one
	return existingIdxRow, retryCount, false, nil
}

func TestIdxNotAppliedDiffPresentLookup(t *testing.T) {
	os.Remove("/tmp/capi_out/lookup_quicktest/order_date_value_grouped_inner.csv")
	os.Remove("/tmp/capi_out/lookup_quicktest/order_date_value_grouped_left_outer.csv")
	os.Remove("/tmp/capi_out/lookup_quicktest/order_item_date_inner.csv")
	os.Remove("/tmp/capi_out/lookup_quicktest/order_item_date_left_outer.csv")

	ks := "ks_idx_not_applied_diff_present_lookup"

	envConfig := env.EnvConfig{
		Cassandra:                         env.CassandraConfig{WriterWorkers: 1},
		Log:                               env.LogConfig{Level: "INFO"},
		CustomProcessorDefFactoryInstance: &TestProcessorDefFactory{},
		CustomProcessorsSettings:          getTestProcessorSettings(),
		UseGocqlmem:                       true,
	}
	sc.ScriptDefCache = sc.NewScriptDefCache()
	NodeDependencyReadynessCache = NewNodeDependencyReadynessCache()

	logger, err := l.NewLoggerFromEnvConfig(&envConfig, "unittest")
	assert.Nil(t, err)
	logger.PushF("TestIdxSeriousErrorLookup")
	defer logger.PopF()

	mqProducer := mq.TestInmemProducer{}

	gocqlmemSession, cassandraEngineType, err := db.NewSession(&envConfig, ks, db.CreateKeyspaceOnConnect)
	assert.Nil(t, err)

	_, err = StartRun(&envConfig, logger, &mqProducer, "/tmp/capi_cfg/lookup_quicktest/script_quick.yaml", "/tmp/capi_cfg/lookup_quicktest/script_params_quick_fs_one.yaml", gocqlmemSession, cassandraEngineType, ks, []string{"read_orders", "read_order_items"}, "test run")
	assert.Nil(t, err)

	var runStatus wfmodel.RunStatusType

	// Verify run status
	runHistory, err := GetRunHistory(gocqlmemSession, ks)
	assert.Nil(t, err)
	runStatus = runHistory[len(runHistory)-1].Status
	assert.Equal(t, wfmodel.RunStart, runStatus)

	for {
		msg := mqProducer.PeekHead()
		if msg == nil {
			break
		}
		queryPerformer := TableInserterQueryPerformerTestIdxNotAppliedDiffPresent{}
		ackCmd := ProcessDataBatchMsg(&envConfig, logger, msg, 0, nil, ctx.TableInserterProperties{
			QueryPerformer:               &queryPerformer,
			DoesNotExistPauseMillis:      10, // speed it up for testing
			OperationTimedOutPauseMillis: 10, // speed it up for testing
		})
		if ackCmd == mq.AcknowledgerCmdAck {
			mqProducer.RemoveHead()
		} else {
			mqProducer.MoveHeadToTail()
		}
	}

	// Verify run status
	runHistory, err = GetRunHistory(gocqlmemSession, ks)
	assert.Nil(t, err)
	runStatus = runHistory[len(runHistory)-1].Status
	assert.Equal(t, wfmodel.RunComplete, runStatus)

	// Verify node statuses
	nodeHistory, err := GetNodeHistoryForRuns(gocqlmemSession, ks, []int16{int16(1)})
	assert.Nil(t, err)
	newNodeRunStatusMap := map[string]wfmodel.NodeBatchStatusType{}
	for _, nodeEvent := range nodeHistory {
		newNodeRunStatusMap[nodeEvent.ScriptNode] = nodeEvent.Status
	}

	// For each node, verify batch statuses
	for nodeName, nodeStatus := range newNodeRunStatusMap {
		assert.Equal(t, wfmodel.NodeBatchFail, nodeStatus, fmt.Sprintf("node %s supposed to fail", nodeName))
		// Make sure all batches for this node started then failed
		batchEvents, err := GetBatchHistoryForRunAndNode(gocqlmemSession, ks, int16(1), nodeName)
		assert.Nil(t, err)
		if nodeName == "read_orders" || nodeName == "read_order_items" {
			// These nodes start and fail
			assert.Equal(t, 2, len(batchEvents), nodeName)
			assert.Equal(t, wfmodel.NodeBatchStart, batchEvents[0].Status, nodeName)
			assert.Equal(t, wfmodel.NodeBatchFail, batchEvents[1].Status, nodeName)
			assert.True(t, strings.Contains(batchEvents[1].Comment, "rowid is different, throwing error duplicate key"))
		} else {
			// These nodes failed without starting
			for _, event := range batchEvents {
				assert.Equal(t, wfmodel.NodeBatchFail, event.Status, nodeName)
				assert.True(t, strings.Contains(event.Comment, "some dependency nodes"))
			}
		}
	}
	assert.Nil(t, gocqlmemSession.Query(fmt.Sprintf("DROP keyspace %s;", ks)).Exec())
	gocqlmemSession.Close()
}

func TestIdxNotAppliedDiffPresentFannieMae(t *testing.T) {
	os.Remove("/tmp/capi_out/fannie_mae_apitest/deal_seller_summaries.csv")
	os.Remove("/tmp/capi_out/fannie_mae_apitest/deal_summaries.csv")
	os.Remove("/tmp/capi_out/fannie_mae_apitest/loan_smrs_clcltd.csv")

	ks := "ks_idx_not_applied_diff_present_fannie_mae"

	envConfig := env.EnvConfig{
		Cassandra:                         env.CassandraConfig{WriterWorkers: 2},
		Log:                               env.LogConfig{Level: "INFO"},
		CustomProcessorDefFactoryInstance: &TestProcessorDefFactory{},
		CustomProcessorsSettings:          getTestProcessorSettings(),
		UseGocqlmem:                       true,
	}
	sc.ScriptDefCache = sc.NewScriptDefCache()
	NodeDependencyReadynessCache = NewNodeDependencyReadynessCache()

	logger, err := l.NewLoggerFromEnvConfig(&envConfig, "unittest")
	assert.Nil(t, err)
	logger.PushF("TestIdxSeriousErrorFannieMae")
	defer logger.PopF()

	mqProducer := mq.TestInmemProducer{}

	gocqlmemSession, cassandraEngineType, err := db.NewSession(&envConfig, ks, db.CreateKeyspaceOnConnect)
	assert.Nil(t, err)

	_, err = StartRun(&envConfig, logger, &mqProducer, "/tmp/capi_cfg/fannie_mae_apitest/script_api.json", "/tmp/capi_cfg/fannie_mae_apitest/script_params_api.json", gocqlmemSession, cassandraEngineType, ks, []string{"01_read_payments"}, "test run")
	assert.Nil(t, err)

	var runStatus wfmodel.RunStatusType

	// Verify run status
	runHistory, err := GetRunHistory(gocqlmemSession, ks)
	assert.Nil(t, err)
	runStatus = runHistory[len(runHistory)-1].Status
	assert.Equal(t, wfmodel.RunStart, runStatus)

	for {
		msg := mqProducer.PeekHead()
		if msg == nil {
			break
		}
		queryPerformer := TableInserterQueryPerformerTestIdxNotAppliedDiffPresent{}
		ackCmd := ProcessDataBatchMsg(&envConfig, logger, msg, 0, nil, ctx.TableInserterProperties{
			QueryPerformer:               &queryPerformer,
			DoesNotExistPauseMillis:      10, // speed it up for testing
			OperationTimedOutPauseMillis: 10, // speed it up for testing
		})
		if ackCmd == mq.AcknowledgerCmdAck {
			mqProducer.RemoveHead()
		} else {
			mqProducer.MoveHeadToTail()
		}
	}

	// Verify run status
	runHistory, err = GetRunHistory(gocqlmemSession, ks)
	assert.Nil(t, err)
	runStatus = runHistory[len(runHistory)-1].Status
	assert.Equal(t, wfmodel.RunComplete, runStatus)

	// Verify node statuses
	nodeHistory, err := GetNodeHistoryForRuns(gocqlmemSession, ks, []int16{int16(1)})
	assert.Nil(t, err)
	newNodeRunStatusMap := map[string]wfmodel.NodeBatchStatusType{}
	for _, nodeEvent := range nodeHistory {
		newNodeRunStatusMap[nodeEvent.ScriptNode] = nodeEvent.Status
	}

	// For each node, verify batch statuses
	for nodeName, nodeStatus := range newNodeRunStatusMap {
		assert.Equal(t, wfmodel.NodeBatchFail, nodeStatus, fmt.Sprintf("node %s supposed to fail", nodeName))
		// Make sure all batches for this node started then failed
		batchEvents, err := GetBatchHistoryForRunAndNode(gocqlmemSession, ks, int16(1), nodeName)
		assert.Nil(t, err)
		if nodeName == "01_read_payments" {
			// These nodes start and fail
			assert.Equal(t, 2, len(batchEvents), nodeName)
			assert.Equal(t, wfmodel.NodeBatchStart, batchEvents[0].Status, nodeName)
			assert.Equal(t, wfmodel.NodeBatchFail, batchEvents[1].Status, nodeName)
			assert.True(t, strings.Contains(batchEvents[1].Comment, "rowid is different, throwing error duplicate key"))
		} else {
			// These nodes failed without starting
			for _, event := range batchEvents {
				assert.Equal(t, wfmodel.NodeBatchFail, event.Status, nodeName)
				assert.True(t, strings.Contains(event.Comment, "some dependency nodes"))
			}
		}
	}
	assert.Nil(t, gocqlmemSession.Query(fmt.Sprintf("DROP keyspace %s;", ks)).Exec())
	gocqlmemSession.Close()
}

// Simulate processor crash

func TestProcesorCrashLookup(t *testing.T) {
	os.Remove("/tmp/capi_out/lookup_quicktest/order_date_value_grouped_inner.csv")
	os.Remove("/tmp/capi_out/lookup_quicktest/order_date_value_grouped_left_outer.csv")
	os.Remove("/tmp/capi_out/lookup_quicktest/order_item_date_inner.csv")
	os.Remove("/tmp/capi_out/lookup_quicktest/order_item_date_left_outer.csv")

	ks := "ks_processor_crash_lookup"

	envConfig := env.EnvConfig{
		Cassandra:                         env.CassandraConfig{WriterWorkers: 1},
		Log:                               env.LogConfig{Level: "INFO"},
		CustomProcessorDefFactoryInstance: &TestProcessorDefFactory{},
		CustomProcessorsSettings:          getTestProcessorSettings(),
		UseGocqlmem:                       true,
	}
	sc.ScriptDefCache = sc.NewScriptDefCache()
	NodeDependencyReadynessCache = NewNodeDependencyReadynessCache()

	logger, err := l.NewLoggerFromEnvConfig(&envConfig, "unittest")
	assert.Nil(t, err)
	logger.PushF("TestProcesorCrashLookup")
	defer logger.PopF()

	mqProducer := mq.TestInmemProducer{}

	gocqlmemSession, cassandraEngineType, err := db.NewSession(&envConfig, ks, db.CreateKeyspaceOnConnect)
	assert.Nil(t, err)

	_, err = StartRun(&envConfig, logger, &mqProducer, "/tmp/capi_cfg/lookup_quicktest/script_quick.yaml", "/tmp/capi_cfg/lookup_quicktest/script_params_quick_fs_one.yaml", gocqlmemSession, cassandraEngineType, ks, []string{"read_orders", "read_order_items"}, "test run")
	assert.Nil(t, err)

	var runStatus wfmodel.RunStatusType

	// Verify run status
	runHistory, err := GetRunHistory(gocqlmemSession, ks)
	assert.Nil(t, err)
	runStatus = runHistory[len(runHistory)-1].Status
	assert.Equal(t, wfmodel.RunStart, runStatus)

	fakeFailedBatchMap := map[string]struct{}{}
	for {
		msg := mqProducer.PeekHead()
		if msg == nil {
			break
		}
		queryPerformer := db.TableInserterQueryPerformerProduction{}
		ackCmd := ProcessDataBatchMsg(&envConfig, logger, msg, 0, nil, ctx.TableInserterProperties{
			QueryPerformer:               &queryPerformer,
			DoesNotExistPauseMillis:      10,  // speed it up for testing
			OperationTimedOutPauseMillis: 100, // speed it up for testing
		})
		if ackCmd == mq.AcknowledgerCmdAck {
			if _, ok := fakeFailedBatchMap[msg.FullBatchId()]; !strings.HasPrefix(msg.TargetNodeName, "file") && !ok {
				fakeFailedBatchMap[msg.FullBatchId()] = struct{}{}
				// Simulate failure and re-process. Rewrite history:
				// - remove batch completed record
				// - remove node complete record
				s, _, err := db.NewSession(&envConfig, msg.DataKeyspace, db.DoNotCreateKeyspaceOnConnect)
				assert.Nil(t, err)

				// Verify the batch marked complete
				batchEvents, err := GetBatchHistoryForRunAndNode(s, msg.DataKeyspace, msg.RunId, msg.TargetNodeName)
				assert.Nil(t, err)

				var batchSuccessful bool
				for _, e := range batchEvents {
					if e.BatchIdx == msg.BatchIdx && e.Status == wfmodel.NodeBatchSuccess {
						batchSuccessful = true
						break
					}
				}
				assert.True(t, batchSuccessful)

				// Rewrite hisory: unmark this batch as complete
				err = s.Query(fmt.Sprintf(`DELETE FROM %s.%s WHERE run_id = %d AND script_node = '%s' AND batch_idx = %d AND status = %d;`, msg.DataKeyspace, wfmodel.TableNameBatchHistory, msg.RunId, msg.TargetNodeName, msg.BatchIdx, wfmodel.NodeBatchSuccess)).Exec()
				assert.Nil(t, err)

				// Was node marked marked complete?
				nodeEvents, err := GetNodeHistoryForRuns(s, msg.DataKeyspace, []int16{msg.RunId})
				assert.Nil(t, err)
				for _, e := range nodeEvents {
					if e.ScriptNode == msg.TargetNodeName && e.Status == wfmodel.NodeBatchSuccess {
						// Node was declared successfully completed, but we will revert it
						err = s.Query(fmt.Sprintf(`DELETE FROM %s.%s WHERE run_id = %d AND script_node = '%s' AND status = %d;`, msg.DataKeyspace, wfmodel.TableNameNodeHistory, msg.RunId, msg.TargetNodeName, wfmodel.NodeBatchSuccess)).Exec()
						assert.Nil(t, err)
						break
					}
				}
				s.Close()

				// Send back to processing again
				mqProducer.MoveHeadToTail()
			} else {
				mqProducer.RemoveHead()
			}
		} else {
			mqProducer.MoveHeadToTail()
		}
	}

	// Verify run status
	runHistory, err = GetRunHistory(gocqlmemSession, ks)
	assert.Nil(t, err)
	runStatus = runHistory[len(runHistory)-1].Status
	assert.Equal(t, wfmodel.RunComplete, runStatus)

	err = compareCsvs("/tmp/capi_out/lookup_quicktest/order_date_value_grouped_inner_baseline.csv", "/tmp/capi_out/lookup_quicktest/order_date_value_grouped_inner.csv")
	assert.Nil(t, err)
	err = compareCsvs("/tmp/capi_out/lookup_quicktest/order_date_value_grouped_left_outer_baseline.csv", "/tmp/capi_out/lookup_quicktest/order_date_value_grouped_left_outer.csv")
	assert.Nil(t, err)
	err = compareCsvs("/tmp/capi_out/lookup_quicktest/order_item_date_inner_baseline.csv", "/tmp/capi_out/lookup_quicktest/order_item_date_inner.csv")
	assert.Nil(t, err)
	err = compareCsvs("/tmp/capi_out/lookup_quicktest/order_item_date_left_outer_baseline.csv", "/tmp/capi_out/lookup_quicktest/order_item_date_left_outer.csv")
	assert.Nil(t, err)

	assert.Nil(t, gocqlmemSession.Query(fmt.Sprintf("DROP keyspace %s;", ks)).Exec())
	gocqlmemSession.Close()
}

func TestProcesorCrashFannieMae(t *testing.T) {
	os.Remove("/tmp/capi_out/fannie_mae_apitest/deal_seller_summaries.csv")
	os.Remove("/tmp/capi_out/fannie_mae_apitest/deal_summaries.csv")
	os.Remove("/tmp/capi_out/fannie_mae_apitest/loan_smrs_clcltd.csv")

	ks := "ks_processor_crash_fannie_mae"

	envConfig := env.EnvConfig{
		Cassandra:                         env.CassandraConfig{WriterWorkers: 2},
		Log:                               env.LogConfig{Level: "INFO"},
		CustomProcessorDefFactoryInstance: &TestProcessorDefFactory{},
		CustomProcessorsSettings:          getTestProcessorSettings(),
		UseGocqlmem:                       true,
	}
	sc.ScriptDefCache = sc.NewScriptDefCache()
	NodeDependencyReadynessCache = NewNodeDependencyReadynessCache()

	logger, err := l.NewLoggerFromEnvConfig(&envConfig, "unittest")
	assert.Nil(t, err)
	logger.PushF("TestIdxSeriousErrorFannieMae")
	defer logger.PopF()

	mqProducer := mq.TestInmemProducer{}

	gocqlmemSession, cassandraEngineType, err := db.NewSession(&envConfig, ks, db.CreateKeyspaceOnConnect)
	assert.Nil(t, err)

	_, err = StartRun(&envConfig, logger, &mqProducer, "/tmp/capi_cfg/fannie_mae_apitest/script_api.json", "/tmp/capi_cfg/fannie_mae_apitest/script_params_api.json", gocqlmemSession, cassandraEngineType, ks, []string{"01_read_payments"}, "test run")
	assert.Nil(t, err)

	var runStatus wfmodel.RunStatusType

	// Verify run status
	runHistory, err := GetRunHistory(gocqlmemSession, ks)
	assert.Nil(t, err)
	runStatus = runHistory[len(runHistory)-1].Status
	assert.Equal(t, wfmodel.RunStart, runStatus)

	fakeFailedBatchMap := map[string]struct{}{}
	for {
		msg := mqProducer.PeekHead()
		if msg == nil {
			break
		}
		queryPerformer := db.TableInserterQueryPerformerProduction{}
		ackCmd := ProcessDataBatchMsg(&envConfig, logger, msg, 0, nil, ctx.TableInserterProperties{
			QueryPerformer:               &queryPerformer,
			DoesNotExistPauseMillis:      10,  // speed it up for testing
			OperationTimedOutPauseMillis: 100, // speed it up for testing
		})
		if ackCmd == mq.AcknowledgerCmdAck {
			if _, ok := fakeFailedBatchMap[msg.FullBatchId()]; !strings.HasPrefix(msg.TargetNodeName, "file") && !ok {
				fakeFailedBatchMap[msg.FullBatchId()] = struct{}{}
				// Simulate failure and re-process. Rewrite history:
				// - remove batch completed record
				// - remove node complete record
				s, _, err := db.NewSession(&envConfig, msg.DataKeyspace, db.DoNotCreateKeyspaceOnConnect)
				assert.Nil(t, err)

				// Verify the batch marked complete
				batchEvents, err := GetBatchHistoryForRunAndNode(s, msg.DataKeyspace, msg.RunId, msg.TargetNodeName)
				assert.Nil(t, err)

				var batchSuccessful bool
				for _, e := range batchEvents {
					if e.BatchIdx == msg.BatchIdx && e.Status == wfmodel.NodeBatchSuccess {
						batchSuccessful = true
						break
					}
				}
				if batchSuccessful {
					// Rewrite hisory: unmark this batch as complete
					err = s.Query(fmt.Sprintf(`DELETE FROM %s.%s WHERE run_id = %d AND script_node = '%s' AND batch_idx = %d AND status = %d;`, msg.DataKeyspace, wfmodel.TableNameBatchHistory, msg.RunId, msg.TargetNodeName, msg.BatchIdx, wfmodel.NodeBatchSuccess)).Exec()
					assert.Nil(t, err)

					// Was node marked marked complete?
					nodeEvents, err := GetNodeHistoryForRuns(s, msg.DataKeyspace, []int16{msg.RunId})
					assert.Nil(t, err)
					for _, e := range nodeEvents {
						if e.ScriptNode == msg.TargetNodeName && e.Status == wfmodel.NodeBatchSuccess {
							// Node was declared successfully completed, but we will revert it
							err = s.Query(fmt.Sprintf(`DELETE FROM %s.%s WHERE run_id = %d AND script_node = '%s' AND status = %d;`, msg.DataKeyspace, wfmodel.TableNameNodeHistory, msg.RunId, msg.TargetNodeName, wfmodel.NodeBatchSuccess)).Exec()
							assert.Nil(t, err)
							break
						}
					}
					s.Close()

					// Send back to processing again
					mqProducer.MoveHeadToTail()
				} else {
					mqProducer.RemoveHead()
				}
			} else {
				mqProducer.RemoveHead()
			}
		} else {
			mqProducer.MoveHeadToTail()
		}
	}

	// Verify run status
	runHistory, err = GetRunHistory(gocqlmemSession, ks)
	assert.Nil(t, err)
	runStatus = runHistory[len(runHistory)-1].Status
	assert.Equal(t, wfmodel.RunComplete, runStatus)

	// Verify node statuses
	nodeHistory, err := GetNodeHistoryForRuns(gocqlmemSession, ks, []int16{int16(1)})
	assert.Nil(t, err)
	newNodeRunStatusMap := map[string]wfmodel.NodeBatchStatusType{}
	for _, nodeEvent := range nodeHistory {
		newNodeRunStatusMap[nodeEvent.ScriptNode] = nodeEvent.Status
	}

	// For each node, verify batch statuses
	for _, nhe := range nodeHistory {
		if nhe.Status != wfmodel.NodeBatchStart {
			switch nhe.ScriptNode {
			case "02_loan_ids", "02_deal_names", "02_deal_sellers", "05_deal_seller_summaries":
				assert.Equal(t, wfmodel.NodeBatchFail, nhe.Status, fmt.Sprintf("node %s supposed to fail", nhe.ScriptNode))
				// Make sure all batches for this node started then failed
				batchEvents, err := GetBatchHistoryForRunAndNode(gocqlmemSession, ks, int16(1), nhe.ScriptNode)
				assert.Nil(t, err)
				for _, be := range batchEvents {
					if be.Status == wfmodel.NodeBatchFail {
						if nhe.ScriptNode == "02_loan_ids" {
							assert.Equal(t, ErrorNotProcessingAbandonedBatch, be.Comment)
						} else {
							assert.Contains(t, be.Comment, "some dependency nodes")
						}
					}
				}
			case "01_read_payments":
				assert.Equal(t, wfmodel.NodeBatchSuccess, nhe.Status)
			}
		}
	}

	assert.Nil(t, gocqlmemSession.Query(fmt.Sprintf("DROP keyspace %s;", ks)).Exec())
	gocqlmemSession.Close()
}
