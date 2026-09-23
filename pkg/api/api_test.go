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
	"time"

	"github.com/capillariesio/gocqlmem/gocqlshims"
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
	"github.com/stretchr/testify/assert"
)

// -----------------------------------------------------------------------------
// CSV comparison helpers
// -----------------------------------------------------------------------------

func openFileWithRetries(filename string) (*os.File, error) {
	for range 5 {
		f, err := os.Open(filename)
		if err == nil {
			return f, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		time.Sleep(time.Duration(1000) * time.Millisecond)
	}
	return nil, os.ErrNotExist
}

func readCSV(filename string) ([][]string, error) {
	f, err := openFileWithRetries(filename)
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

// -----------------------------------------------------------------------------
// Custom processor plumbing shared by all tests
// -----------------------------------------------------------------------------

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

// -----------------------------------------------------------------------------
// Test scenarios
//
// Every test drives one of two scripts. A scenario captures everything that
// differs between them: the script/params files, the nodes the run starts
// with, and the output files to clean up and compare against a baseline.
// -----------------------------------------------------------------------------

type scenario struct {
	scriptPath string
	paramsPath string
	startNodes []string
	outDir     string
	outFiles   []string // basenames without the ".csv" suffix

	// isStartNode reports whether a node is one that actually starts a batch
	// (and therefore, in the failure tests, starts-then-fails) as opposed to a
	// downstream node that fails without ever starting.
	isStartNode func(nodeName string) bool
}

var lookupScenario = scenario{
	scriptPath: "/tmp/capi_cfg/lookup_quicktest/script_quick.yaml",
	paramsPath: "/tmp/capi_cfg/lookup_quicktest/script_params_quick_fs_one.yaml",
	startNodes: []string{"read_orders", "read_order_items"},
	outDir:     "/tmp/capi_out/lookup_quicktest",
	outFiles: []string{
		"order_date_value_grouped_inner",
		"order_date_value_grouped_left_outer",
		"order_item_date_inner",
		"order_item_date_left_outer",
	},
	isStartNode: func(n string) bool { return n == "read_orders" || n == "read_order_items" },
}

var fannieMaeScenario = scenario{
	scriptPath: "/tmp/capi_cfg/fannie_mae_apitest/script_api.json",
	paramsPath: "/tmp/capi_cfg/fannie_mae_apitest/script_params_api.json",
	startNodes: []string{"01_read_payments"},
	outDir:     "/tmp/capi_out/fannie_mae_apitest",
	outFiles: []string{
		"deal_seller_summaries",
		"deal_summaries",
		"loan_smrs_clcltd",
	},
	isStartNode: func(n string) bool { return n == "01_read_payments" },
}

func (s scenario) outPath(base string) string      { return s.outDir + "/" + base + ".csv" }
func (s scenario) baselinePath(base string) string { return s.outDir + "/" + base + "_baseline.csv" }

func (s scenario) removeOutputs() {
	for _, f := range s.outFiles {
		os.Remove(s.outPath(f))
	}
}

func (s scenario) compareOutputs(t *testing.T) {
	for _, f := range s.outFiles {
		err := compareCsvs(s.baselinePath(f), s.outPath(f))
		assert.Nil(t, err, fmt.Sprintf("%v", err))
	}
}

// -----------------------------------------------------------------------------
// Test run harness
//
// testRun owns the boilerplate that every test repeats: cleaning outputs,
// building the env config, resetting caches, opening a session, kicking off the
// run and draining the message queue.
// -----------------------------------------------------------------------------

type testRun struct {
	t       *testing.T
	scn     scenario
	ks      string
	ec      env.EnvConfig
	logger  *l.CapiLogger
	prod    *mq.TestInmemProducer
	session gocqlshims.Session
}

// startTestRun cleans outputs, starts a run for the given scenario and asserts
// it entered RunStart. Callers should `defer r.cleanup()`.
func startTestRun(t *testing.T, scn scenario, ksSuffix string, writerWorkers int) *testRun {
	scn.removeOutputs()

	ks := "ks_" + ksSuffix
	ec := env.EnvConfig{
		Cassandra:                         env.CassandraConfig{WriterWorkers: writerWorkers},
		Log:                               env.LogConfig{Level: "INFO"},
		CustomProcessorDefFactoryInstance: &TestProcessorDefFactory{},
		CustomProcessorsSettings:          getTestProcessorSettings(),
		UseGocqlmem:                       true,
	}
	sc.ScriptDefCache = sc.NewScriptDefCache()
	NodeDependencyReadynessCache = NewNodeDependencyReadynessCache()

	logger, err := l.NewLoggerFromEnvConfig(&ec, "unittest")
	assert.Nil(t, err, fmt.Sprintf("%v", err))
	logger.PushF(t.Name())

	prod := &mq.TestInmemProducer{}

	session, engineType, err := db.NewSession(&ec, ks, db.CreateKeyspaceOnConnect)
	assert.Nil(t, err, fmt.Sprintf("%v", err))

	_, err = StartRun(&ec, logger, prod, scn.scriptPath, scn.paramsPath, session, engineType, ks, scn.startNodes, "test run")
	assert.Nil(t, err, fmt.Sprintf("%v", err))

	r := &testRun{t: t, scn: scn, ks: ks, ec: ec, logger: logger, prod: prod, session: session}
	r.assertRunStatus(wfmodel.RunStart)
	return r
}

func (r *testRun) cleanup() {
	assert.Nil(r.t, r.session.Query(fmt.Sprintf("DROP keyspace %s;", r.ks)).Exec())
	r.session.Close()
	r.logger.PopF()
}

func (r *testRun) assertRunStatus(expected wfmodel.RunStatusType) {
	runHistory, err := GetRunHistory(r.session, r.ks)
	assert.Nil(r.t, err, fmt.Sprintf("%v", err))
	assert.Equal(r.t, expected, runHistory[len(runHistory)-1].Status)
}

// finish asserts the run reached RunComplete.
func (r *testRun) finish() { r.assertRunStatus(wfmodel.RunComplete) }

// verifyFunc runs per-message assertions after a batch has been processed.
type verifyFunc func(msg *wfmodel.Message)

// processAll drains the queue, processing every batch with a fresh query
// performer. newHandler returns that performer together with an optional
// verifyFunc (may be nil) that inspects the performer after processing.
func (r *testRun) processAll(newHandler func() (db.TableInserterQueryPerformer, verifyFunc)) {
	for {
		msg := r.prod.PeekHead()
		if msg == nil {
			break
		}
		qp, verify := newHandler()
		ackCmd := ProcessDataBatchMsg(&r.ec, r.logger, msg, 0, nil, ctx.TableInserterProperties{
			QueryPerformer:               qp,
			DoesNotExistPauseMillis:      10,  // speed it up for testing
			OperationTimedOutPauseMillis: 100, // speed it up for testing
		})
		if ackCmd == mq.AcknowledgerCmdAck {
			r.prod.RemoveHead()
		} else {
			r.prod.MoveHeadToTail()
		}
		if verify != nil {
			verify(msg)
		}
	}
}

// assertSimulatedErrorCount checks that a simulated error was (or was not)
// introduced for a node: exactly one when expected, none otherwise.
func assertSimulatedErrorCount(t *testing.T, msg *wfmodel.Message, actual int, expectError bool) {
	want := 0
	if expectError {
		want = 1
	}
	assert.Equal(t, want, actual, msg.TargetNodeName)
}

// assertAllNodesFailed verifies the failure-mode expectations shared by every
// "serious error" / "not applied" test: every node ends in NodeBatchFail. Nodes
// that actually started go Start->Fail with failComment; downstream nodes fail
// without starting, blamed on "some dependency nodes".
func (r *testRun) assertAllNodesFailed(failComment string) {
	t := r.t
	nodeHistory, err := GetNodeHistoryForRuns(r.session, r.ks, []int16{int16(1)})
	assert.Nil(t, err, fmt.Sprintf("%v", err))

	nodeStatus := map[string]wfmodel.NodeBatchStatusType{}
	for _, nodeEvent := range nodeHistory {
		nodeStatus[nodeEvent.ScriptNode] = nodeEvent.Status
	}

	for nodeName, status := range nodeStatus {
		assert.Equal(t, wfmodel.NodeBatchFail, status, fmt.Sprintf("node %s supposed to fail", nodeName))
		batchEvents, err := GetBatchHistoryForRunAndNode(r.session, r.ks, int16(1), nodeName)
		assert.Nil(t, err, fmt.Sprintf("%v", err))
		if r.scn.isStartNode(nodeName) {
			// These nodes start and fail
			assert.Equal(t, 2, len(batchEvents), nodeName)
			assert.Equal(t, wfmodel.NodeBatchStart, batchEvents[0].Status, nodeName)
			assert.Equal(t, wfmodel.NodeBatchFail, batchEvents[1].Status, nodeName)
			assert.True(t, strings.Contains(batchEvents[1].Comment, failComment))
		} else {
			// These nodes failed without starting
			for _, event := range batchEvents {
				assert.Equal(t, wfmodel.NodeBatchFail, event.Status, nodeName)
				assert.True(t, strings.Contains(event.Comment, "some dependency nodes"))
			}
		}
	}
}

// processAllWithSimulatedCrash drains the queue using the production query
// performer, but for the first ack of each non-file batch it rewrites history to
// pretend the processor crashed after committing, then re-queues the batch. This
// exercises re-processing of already-completed batches.
func (r *testRun) processAllWithSimulatedCrash(isWriteFileNode func(nodeName string) bool, requireBatchSuccess bool) {
	fakeFailedBatchMap := map[string]struct{}{}
	for {
		msg := r.prod.PeekHead()
		if msg == nil {
			break
		}
		queryPerformer := db.TableInserterQueryPerformerProduction{}
		ackCmd := ProcessDataBatchMsg(&r.ec, r.logger, msg, 0, nil, ctx.TableInserterProperties{
			QueryPerformer:               &queryPerformer,
			DoesNotExistPauseMillis:      10,  // speed it up for testing
			OperationTimedOutPauseMillis: 100, // speed it up for testing
		})
		if ackCmd != mq.AcknowledgerCmdAck {
			r.prod.MoveHeadToTail()
			continue
		}

		_, alreadyFailed := fakeFailedBatchMap[msg.FullBatchId()]
		if isWriteFileNode(msg.TargetNodeName) || alreadyFailed {
			r.prod.RemoveHead()
			continue
		}

		fakeFailedBatchMap[msg.FullBatchId()] = struct{}{}
		if r.rewriteHistoryToSimulateCrash(msg, requireBatchSuccess) {
			// Send back to processing again
			r.prod.MoveHeadToTail()
		} else {
			r.prod.RemoveHead()
		}
	}
}

// rewriteHistoryToSimulateCrash removes the batch-complete and node-complete
// records for msg's batch so it will be re-processed. It reports whether the
// batch had actually been marked complete. When requireBatchSuccess is set, that
// is additionally asserted.
func (r *testRun) rewriteHistoryToSimulateCrash(msg *wfmodel.Message, requireBatchSuccess bool) bool {
	t := r.t
	s, _, err := db.NewSession(&r.ec, msg.DataKeyspace, db.DoNotCreateKeyspaceOnConnect)
	assert.Nil(t, err, fmt.Sprintf("%v", err))
	defer s.Close()

	// Verify the batch marked complete
	batchEvents, err := GetBatchHistoryForRunAndNode(s, msg.DataKeyspace, msg.RunId, msg.TargetNodeName)
	assert.Nil(t, err, fmt.Sprintf("%v", err))

	var batchSuccessful bool
	for _, e := range batchEvents {
		if e.BatchIdx == msg.BatchIdx && e.Status == wfmodel.NodeBatchSuccess {
			batchSuccessful = true
			break
		}
	}
	if requireBatchSuccess {
		assert.True(t, batchSuccessful)
	}
	if !batchSuccessful {
		return false
	}

	// Rewrite hisory: unmark this batch as complete
	err = s.Query(fmt.Sprintf(`DELETE FROM %s.%s WHERE run_id = %d AND script_node = '%s' AND batch_idx = %d AND status = %d;`, msg.DataKeyspace, wfmodel.TableNameBatchHistory, msg.RunId, msg.TargetNodeName, msg.BatchIdx, wfmodel.NodeBatchSuccess)).Exec()
	assert.Nil(t, err, fmt.Sprintf("%v", err))

	// Was node marked marked complete?
	nodeEvents, err := GetNodeHistoryForRuns(s, msg.DataKeyspace, []int16{msg.RunId})
	assert.Nil(t, err, fmt.Sprintf("%v", err))
	for _, e := range nodeEvents {
		if e.ScriptNode == msg.TargetNodeName && e.Status == wfmodel.NodeBatchSuccess {
			// Node was declared successfully completed, but we will revert it
			err = s.Query(fmt.Sprintf(`DELETE FROM %s.%s WHERE run_id = %d AND script_node = '%s' AND status = %d;`, msg.DataKeyspace, wfmodel.TableNameNodeHistory, msg.RunId, msg.TargetNodeName, wfmodel.NodeBatchSuccess)).Exec()
			assert.Nil(t, err, fmt.Sprintf("%v", err))
			break
		}
	}
	return true
}

// -----------------------------------------------------------------------------
// Table does not exist: first data/idx write reports "does not exist", retry succeeds
// -----------------------------------------------------------------------------

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
	r := startTestRun(t, lookupScenario, "table_does_not_exist_lookup", 2)
	defer r.cleanup()

	r.processAll(func() (db.TableInserterQueryPerformer, verifyFunc) {
		qp := &TableInserterQueryPerformerTestDataAndIdxDoesNotExist{}
		return qp, func(msg *wfmodel.Message) {
			if qp.TotalDataHits > 0 {
				assertSimulatedErrorCount(t, msg, qp.SimulatedDataErrorCount, !strings.HasPrefix(msg.TargetNodeName, "file_"))
			}
			if qp.TotalIdxHits > 0 {
				assertSimulatedErrorCount(t, msg, qp.SimulatedIdxErrorCount, strings.HasPrefix(msg.TargetNodeName, "read_"))
			}
		}
	})

	r.finish()
	r.scn.compareOutputs(t)
}

func TestTableDoesNotExistFannieMae(t *testing.T) {
	r := startTestRun(t, fannieMaeScenario, "table_does_not_exist_fannie_mae", 4)
	defer r.cleanup()

	r.processAll(func() (db.TableInserterQueryPerformer, verifyFunc) {
		qp := &TableInserterQueryPerformerTestDataAndIdxDoesNotExist{}
		return qp, func(msg *wfmodel.Message) {
			if qp.TotalDataHits > 0 {
				assertSimulatedErrorCount(t, msg, qp.SimulatedDataErrorCount, !strings.Contains(msg.TargetNodeName, "write_file_"))
			}
			if qp.TotalIdxHits > 0 {
				isIdxErrorNode := strings.HasPrefix(msg.TargetNodeName, "01_") || strings.HasPrefix(msg.TargetNodeName, "02_") || msg.TargetNodeName == "04_loan_smrs_clcltd"
				assertSimulatedErrorCount(t, msg, qp.SimulatedIdxErrorCount, isIdxErrorNode)
			}
		}
	})

	r.finish()
	r.scn.compareOutputs(t)
}

// -----------------------------------------------------------------------------
// OperationTimedOut: first data/idx write times out, retry succeeds
// -----------------------------------------------------------------------------

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
	r := startTestRun(t, lookupScenario, "operation_timed_out_lookup", 2)
	defer r.cleanup()

	r.processAll(func() (db.TableInserterQueryPerformer, verifyFunc) {
		qp := &TableInserterQueryPerformerTestOperationTimedOut{}
		return qp, func(msg *wfmodel.Message) {
			if qp.TotalDataHits > 0 {
				assertSimulatedErrorCount(t, msg, qp.SimulatedDataErrorCount, !strings.Contains(msg.TargetNodeName, "write_file_"))
			}
			if qp.TotalIdxHits > 0 {
				assertSimulatedErrorCount(t, msg, qp.SimulatedIdxErrorCount, strings.HasPrefix(msg.TargetNodeName, "read_"))
			}
		}
	})

	r.finish()
	r.scn.compareOutputs(t)
}

func TestOperationTimedOutFannieMae(t *testing.T) {
	r := startTestRun(t, fannieMaeScenario, "operation_timed_out_fannie_mae", 2)
	defer r.cleanup()

	r.processAll(func() (db.TableInserterQueryPerformer, verifyFunc) {
		qp := &TableInserterQueryPerformerTestOperationTimedOut{}
		return qp, func(msg *wfmodel.Message) {
			if qp.TotalDataHits > 0 {
				assertSimulatedErrorCount(t, msg, qp.SimulatedDataErrorCount, !strings.Contains(msg.TargetNodeName, "write_file_"))
			}
			if qp.TotalIdxHits > 0 {
				isIdxErrorNode := strings.HasPrefix(msg.TargetNodeName, "01_") || strings.HasPrefix(msg.TargetNodeName, "02_") || msg.TargetNodeName == "04_loan_smrs_clcltd"
				assertSimulatedErrorCount(t, msg, qp.SimulatedIdxErrorCount, isIdxErrorNode)
			}
		}
	})

	r.finish()
	r.scn.compareOutputs(t)
}

// -----------------------------------------------------------------------------
// Data serious error: first data write hits a fatal error, run fails
// -----------------------------------------------------------------------------

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
	r := startTestRun(t, lookupScenario, "data_serious_error_lookup", 1)
	defer r.cleanup()

	r.processAll(func() (db.TableInserterQueryPerformer, verifyFunc) {
		return &TableInserterQueryPerformerTestDataSeriousError{}, nil
	})

	r.finish()
	r.assertAllNodesFailed(cql.ErrorSomeSeriousError)
}

func TestDataSeriousErrorFannieMae(t *testing.T) {
	r := startTestRun(t, fannieMaeScenario, "data_serious_error_fannie_mae", 1)
	defer r.cleanup()

	r.processAll(func() (db.TableInserterQueryPerformer, verifyFunc) {
		return &TableInserterQueryPerformerTestDataSeriousError{}, nil
	})

	r.finish()
	r.assertAllNodesFailed(cql.ErrorSomeSeriousError)
}

// -----------------------------------------------------------------------------
// Idx serious error: first idx write hits a fatal error, run fails
// -----------------------------------------------------------------------------

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
	r := startTestRun(t, lookupScenario, "idx_serious_error_lookup", 1)
	defer r.cleanup()

	r.processAll(func() (db.TableInserterQueryPerformer, verifyFunc) {
		return &TableInserterQueryPerformerTestIdxSeriousError{}, nil
	})

	r.finish()
	r.assertAllNodesFailed(cql.ErrorSomeSeriousError)
}

func TestIdxSeriousErrorFannieMae(t *testing.T) {
	r := startTestRun(t, fannieMaeScenario, "idx_serious_error_fannie_mae", 1)
	defer r.cleanup()

	r.processAll(func() (db.TableInserterQueryPerformer, verifyFunc) {
		return &TableInserterQueryPerformerTestIdxSeriousError{}, nil
	})

	r.finish()
	r.assertAllNodesFailed(cql.ErrorSomeSeriousError)
}

// -----------------------------------------------------------------------------
// Data not applied: first data write is a non-fatal duplicate rowid, retry with
// a new rowid succeeds
// -----------------------------------------------------------------------------

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
	r := startTestRun(t, lookupScenario, "data_not_applied_lookup", 2)
	defer r.cleanup()

	r.processAll(func() (db.TableInserterQueryPerformer, verifyFunc) {
		qp := &TableInserterQueryPerformerTestDataNotApplied{}
		return qp, func(msg *wfmodel.Message) {
			if qp.TotalDataHits > 0 {
				assertSimulatedErrorCount(t, msg, qp.SimulatedDataErrorCount, !strings.HasPrefix(msg.TargetNodeName, "file_"))
			}
		}
	})

	r.finish()
	r.scn.compareOutputs(t)
}

func TestDataNotAppliedFannieMae(t *testing.T) {
	r := startTestRun(t, fannieMaeScenario, "data_not_applied_fannie_mae", 2)
	defer r.cleanup()

	r.processAll(func() (db.TableInserterQueryPerformer, verifyFunc) {
		qp := &TableInserterQueryPerformerTestDataNotApplied{}
		return qp, func(msg *wfmodel.Message) {
			if qp.TotalDataHits > 0 {
				assertSimulatedErrorCount(t, msg, qp.SimulatedDataErrorCount, !strings.Contains(msg.TargetNodeName, "write_file"))
			}
		}
	})

	r.finish()
	r.scn.compareOutputs(t)
}

// -----------------------------------------------------------------------------
// IdxNotAppliedSamePresentFirstRetry
// This exercises "cannot write duplicate index key "
// -----------------------------------------------------------------------------

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
	r := startTestRun(t, lookupScenario, "idx_not_applied_same_present_first_retry_lookup", 1)
	defer r.cleanup()

	r.processAll(func() (db.TableInserterQueryPerformer, verifyFunc) {
		return &TableInserterQueryPerformerTestIdxNotAppliedSamePresentFirstRetry{}, nil
	})

	r.finish()
	r.assertAllNodesFailed("cannot write duplicate index key ")
}

func TestIdxNotAppliedSamePresentFirstRetryFannieMae(t *testing.T) {
	r := startTestRun(t, fannieMaeScenario, "idx_not_applied_same_present_first_retry_fannie_mae", 1)
	defer r.cleanup()

	r.processAll(func() (db.TableInserterQueryPerformer, verifyFunc) {
		return &TableInserterQueryPerformerTestIdxNotAppliedSamePresentFirstRetry{}, nil
	})

	r.finish()
	r.assertAllNodesFailed("cannot write duplicate index key ")
}

// -----------------------------------------------------------------------------
// IdxNotAppliedSamePresentSecondRetry: not-applied on the first retry, succeeds
// on the second (non-unique idx re-drive)
// -----------------------------------------------------------------------------

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
	r := startTestRun(t, lookupScenario, "idx_not_applied_same_present_second_retry_lookup", 2)
	defer r.cleanup()

	r.processAll(func() (db.TableInserterQueryPerformer, verifyFunc) {
		return &TableInserterQueryPerformerTestIdxNotAppliedSamePresentSecondRetry{}, nil
	})

	r.finish()
	r.scn.compareOutputs(t)
}

func TestIdxNotAppliedSamePresentSecondRetryFannieMae(t *testing.T) {
	r := startTestRun(t, fannieMaeScenario, "idx_not_applied_same_present_second_retry_fannie_mae", 1)
	defer r.cleanup()

	r.processAll(func() (db.TableInserterQueryPerformer, verifyFunc) {
		return &TableInserterQueryPerformerTestIdxNotAppliedSamePresentSecondRetry{}, nil
	})

	r.finish()
	r.scn.compareOutputs(t)
}

// -----------------------------------------------------------------------------
// TestIdxNotAppliedDiffPresent
// This exercises "cannot write index key "
// -----------------------------------------------------------------------------

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
	r := startTestRun(t, lookupScenario, "idx_not_applied_diff_present_lookup", 1)
	defer r.cleanup()

	r.processAll(func() (db.TableInserterQueryPerformer, verifyFunc) {
		return &TableInserterQueryPerformerTestIdxNotAppliedDiffPresent{}, nil
	})

	r.finish()
	r.assertAllNodesFailed("cannot write duplicate unique index key")
}

func TestIdxNotAppliedDiffPresentFannieMae(t *testing.T) {
	r := startTestRun(t, fannieMaeScenario, "idx_not_applied_diff_present_fannie_mae", 2)
	defer r.cleanup()

	r.processAll(func() (db.TableInserterQueryPerformer, verifyFunc) {
		return &TableInserterQueryPerformerTestIdxNotAppliedDiffPresent{}, nil
	})

	r.finish()
	r.assertAllNodesFailed("cannot write duplicate unique index key")
}

// -----------------------------------------------------------------------------
// Simulate processor crash: re-process batches that were already committed
// -----------------------------------------------------------------------------

func TestProcesorCrashLookup(t *testing.T) {
	r := startTestRun(t, lookupScenario, "processor_crash_lookup", 1)
	defer r.cleanup()

	r.processAllWithSimulatedCrash(func(n string) bool { return strings.HasPrefix(n, "file") }, true)

	r.finish()
	r.scn.compareOutputs(t)
}

func TestProcesorCrashFannieMae(t *testing.T) {
	r := startTestRun(t, fannieMaeScenario, "processor_crash_fannie_mae", 2)
	defer r.cleanup()

	r.processAllWithSimulatedCrash(func(n string) bool { return strings.Contains(n, "write_file") }, false)

	r.finish()
	// Node-status verification is intentionally skipped here: with re-driven
	// batches the per-node fail/success pattern is nondeterministic, so we only
	// assert the run completes and the outputs match the baseline.
	r.scn.compareOutputs(t)
}
