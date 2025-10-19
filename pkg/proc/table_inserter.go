package proc

import (
	"errors"
	"fmt"
	"hash/fnv"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/capillariesio/capillaries/pkg/cql"
	"github.com/capillariesio/capillaries/pkg/ctx"
	"github.com/capillariesio/capillaries/pkg/db"
	"github.com/capillariesio/capillaries/pkg/env"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/sc"
)

type DataIdxSeqModeType int

const (
	DataIdxSeqModeDataFirst        DataIdxSeqModeType = iota
	DataIdxSeqModeDistinctIdxFirst                    // Tells us to use idx as a uniqness vehicle for Distinct processor
)

type TableInserter struct {
	PCtx                         *ctx.MessageProcessingContext
	TableCreator                 *sc.TableCreatorDef
	RecordsIn                    chan WriteChannelItem // Channel to pass records from the main function like RunCreateTableForBatch, usig add(), to TableInserter
	RecordWrittenStatuses        chan error
	RecordWrittenStatusesMutex   sync.Mutex // Only to report on draining, otherwise useless
	MachineHash                  int64
	NumWorkers                   int
	DrainerCapacity              int
	WorkerWaitGroup              sync.WaitGroup
	RecordsSent                  int   // Records sent to RecordsIn
	RecordsProcessed             int   // Number of items received in RecordWrittenStatuses
	DoesNotExistPauseMillis      int64 // millis
	OperationTimedOutPauseMillis int64 // millis
	ExpBackoffFactorMultiplier   int64 // 2
	MaxDbProblemRetries          int   // 5
	MaxDuplicateRetries          int   // 5
	DataIdxSeqMode               DataIdxSeqModeType
	MaxAllowedRowInsertionTimeMs int64
	DrainerCancelSignal          chan error
	DrainerCompleteSignal        chan error
	DrainerDoneSignal            chan error
}

type TableRecordItem struct {
	FieldName string
	Value     any
}

func buildTableRecordItems(tr TableRecord) []TableRecordItem {
	result := make([]TableRecordItem, len(tr))
	i := 0
	for fieldName, val := range tr {
		result[i] = TableRecordItem{fieldName, val}
		i++
	}
	return result
}

type IndexKeyItem struct {
	IdxName  string
	KeyValue string
}

func buildIndexKeyItems(ikm map[string]string) []IndexKeyItem {
	result := make([]IndexKeyItem, len(ikm))
	i := 0
	for idxName, keyVal := range ikm {
		result[i] = IndexKeyItem{idxName, keyVal}
		i++
	}
	return result
}

type WriteChannelItem struct {
	TableRecordItems []TableRecordItem
	IndexKeyItems    []IndexKeyItem
}

func (wci *WriteChannelItem) findIndexKeyValByName(idxName string) (string, error) {
	for _, ikmi := range wci.IndexKeyItems {
		if idxName == ikmi.IdxName {
			return ikmi.KeyValue, nil
		}
	}
	return "", fmt.Errorf("cannot find idx name %s in %v", idxName, wci.IndexKeyItems)
}

var seedCounter = int64(0)

func newSeed(hash int64) int64 {
	return hash + atomic.AddInt64(&seedCounter, 1)
}

func createInserterAndStartWorkers(logger *l.CapiLogger, envConfig *env.EnvConfig, pCtx *ctx.MessageProcessingContext, tableCreator *sc.TableCreatorDef, dataIdxSeqMode DataIdxSeqModeType, stringForHash string) (*TableInserter, error) {
	logger.PushF("proc.createInserterAndStartWorkers/TableInserter")
	defer logger.PopF()

	stringHash := fnv.New64a()
	if _, err := stringHash.Write([]byte(stringForHash)); err != nil {
		return nil, err
	}

	instr := &TableInserter{
		PCtx:                         pCtx,
		TableCreator:                 tableCreator,
		RecordsIn:                    make(chan WriteChannelItem, envConfig.Cassandra.WriterWorkers),
		RecordWrittenStatuses:        make(chan error, envConfig.Cassandra.WriterWorkers),
		MachineHash:                  int64(stringHash.Sum64()),
		NumWorkers:                   envConfig.Cassandra.WriterWorkers,
		RecordsSent:                  0,    // Total number of records added to RecordsIn
		RecordsProcessed:             0,    // Total number of records read from RecordWrittenStatuses
		DoesNotExistPauseMillis:      2000, // 2000 + 4000 + 8000 + 16000 + 32000
		OperationTimedOutPauseMillis: 200,  // millis 200 + 400 + 800 + 1600 + 3200 = 6200
		ExpBackoffFactorMultiplier:   2,
		MaxDbProblemRetries:          5,
		MaxDuplicateRetries:          5,
		DataIdxSeqMode:               dataIdxSeqMode,
		DrainerCancelSignal:          make(chan error, 1),
		DrainerCompleteSignal:        make(chan error, 1),
		DrainerDoneSignal:            make(chan error, 1),
	}
	maxInsertionTimeForDoesNotExistMs := cql.SumOfExpBackoffDelaysMs(instr.DoesNotExistPauseMillis, instr.ExpBackoffFactorMultiplier, instr.MaxDbProblemRetries)
	maxInsertionTimeForOperationTimeoutMs := cql.SumOfExpBackoffDelaysMs(instr.OperationTimedOutPauseMillis, instr.ExpBackoffFactorMultiplier, instr.MaxDbProblemRetries)
	instr.MaxAllowedRowInsertionTimeMs = maxInsertionTimeForDoesNotExistMs
	if instr.MaxAllowedRowInsertionTimeMs < maxInsertionTimeForOperationTimeoutMs {
		instr.MaxAllowedRowInsertionTimeMs = maxInsertionTimeForOperationTimeoutMs
	}

	logger.DebugCtx(pCtx, "launching %d writers...", instr.NumWorkers)

	for w := 0; w < instr.NumWorkers; w++ {
		newLogger, err := l.NewLoggerFromLogger(logger)
		if err != nil {
			return nil, err
		}
		// Increase busy worker count
		instr.WorkerWaitGroup.Add(1)
		go instr.tableInserterWorker(newLogger, pCtx)
	}

	logger.DebugCtx(pCtx, "launched %d writers", instr.NumWorkers)

	return instr, nil

}

func CreateDataTableCql(keyspace string, runId int16, tableCreator *sc.TableCreatorDef) string {
	qb := cql.NewQB()
	qb.ColumnDef("rowid", sc.FieldTypeInt)
	qb.ColumnDef("batch_idx", sc.FieldTypeInt)
	for fieldName, fieldDef := range tableCreator.Fields {
		qb.ColumnDef(fieldName, fieldDef.Type)
	}
	return qb.PartitionKey("rowid").Keyspace(keyspace).CreateRun(tableCreator.Name, runId, cql.IgnoreIfExists, tableCreator.CreateProperties)
}

func CreateIdxTableCql(keyspace string, runId int16, idxName string, idxDef *sc.IdxDef, tableCreator *sc.TableCreatorDef) string {
	qb := cql.NewQB()
	qb.Keyspace(keyspace).
		ColumnDef("key", sc.FieldTypeString).
		ColumnDef("rowid", sc.FieldTypeInt)
	if idxDef.Uniqueness == sc.IdxUnique {
		// Key must be unique, let Cassandra enforce it for us: PRIMARY KEY (key)
		qb.PartitionKey("key")
	} else {
		// There can be multiple rowids with the same key:  PRIMARY KEY (key, rowid)
		qb.PartitionKey("key")
		qb.ClusteringKey("rowid")
	}
	return qb.CreateRun(idxName, runId, cql.IgnoreIfExists, tableCreator.CreateProperties)
}

type InserterDrainStrategy int

const (
	InserterDrainCompletely InserterDrainStrategy = iota
	InserterDrainSome
)

const MaxInserterErrors int = 5

func (instr *TableInserter) startDrainer() {
	go func() {
		errorsFound := make([]string, 0)
		stillSending := true
		for stillSending || instr.RecordsSent > instr.RecordsProcessed {
			// Read from instr.RecordWrittenStatuses with timeout (just in case those Cassandra timeouts are not reliable enough)
			timeoutChannel := make(chan bool, 1)
			go func() {
				time.Sleep(time.Duration(instr.MaxAllowedRowInsertionTimeMs) * time.Millisecond)
				timeoutChannel <- true
			}()
			var err error
			select {
			case err = <-instr.RecordWrittenStatuses:
				instr.RecordsProcessed++
				if err != nil {
					if len(errorsFound) < MaxInserterErrors {
						errorsFound = append(errorsFound, err.Error())
					}
				}
			case <-timeoutChannel:
				err = fmt.Errorf("got a timeout while draining, records sent %d, processed %d", instr.RecordsSent, instr.RecordsProcessed)
				errorsFound = append(errorsFound, err.Error())
			case err = <-instr.DrainerCancelSignal:
				errorsFound = append(errorsFound, err.Error())
			case <-instr.DrainerDoneSignal:
				// Now select all while instr.RecordsSent > instr.RecordsProcessed and finish
				stillSending = false
			}
			if err != nil && len(errorsFound) == MaxInserterErrors {
				errorsFound = append(errorsFound, "too many errors in TableInserter")
			}
		}
		if len(errorsFound) > 0 {
			instr.DrainerCompleteSignal <- errors.New(strings.Join(errorsFound, "; "))
		} else {
			instr.DrainerCompleteSignal <- nil
		}
	}()
}

func (instr *TableInserter) doneSending() {
	instr.DrainerDoneSignal <- nil
}

func (instr *TableInserter) cancelDrainer(err error) {
	instr.DrainerCancelSignal <- err
}

func (instr *TableInserter) waitForDrainer(logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext) error {
	err := <-instr.DrainerCompleteSignal // This error will hold the result of all harvested writers
	if err != nil {
		logger.ErrorCtx(pCtx, "error(s) while waiting for workers to drain RecordsIn: %s", err.Error())
	}
	return err
}

func (instr *TableInserter) closeInserter(logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext) {

	// Close instr.RecordsIn, so workers can get out of the "for writeItem := range instr.RecordsIn" loop
	logger.DebugCtx(pCtx, "closing RecordsIn")
	close(instr.RecordsIn)
	logger.DebugCtx(pCtx, "closed RecordsIn")

	// Workers complete
	logger.DebugCtx(pCtx, "waiting for writer workers to complete...")
	instr.WorkerWaitGroup.Wait()
	logger.DebugCtx(pCtx, "writer workers are done")

	// Now it's safe to close RecordWrittenStatuses
	logger.DebugCtx(pCtx, "closing RecordWrittenStatuses")
	close(instr.RecordWrittenStatuses)
	logger.DebugCtx(pCtx, "closed RecordWrittenStatuses")

	logger.DebugCtx(pCtx, "closing DrainerCancelSignal")
	close(instr.DrainerCancelSignal)
	logger.DebugCtx(pCtx, "closed DrainerCancelSignal")

	logger.DebugCtx(pCtx, "closing DrainerCompleteSignal")
	close(instr.DrainerCompleteSignal)
	logger.DebugCtx(pCtx, "closed DrainerCompleteSignal")
}

func (instr *TableInserter) buildIndexKeys(tableRecord TableRecord, indexKeyMap map[string]string) error {
	clear(indexKeyMap)
	for idxName, idxDef := range instr.TableCreator.Indexes {
		var err error
		indexKeyMap[idxName], err = sc.BuildKey(tableRecord, idxDef)
		if err != nil {
			return fmt.Errorf("cannot build key for idx %s, table record [%v]: [%s]", idxName, tableRecord, err.Error())
		}
	}

	return nil
}

func (instr *TableInserter) add(tableRecord TableRecord, indexKeyMap map[string]string) {

	// Do not reuse maps, make GC's job easier
	instr.RecordsIn <- WriteChannelItem{TableRecordItems: buildTableRecordItems(tableRecord), IndexKeyItems: buildIndexKeyItems(indexKeyMap)}
	instr.RecordsSent++
}

func newDataQueryBuilder(keyspace string, tableRecordItems []TableRecordItem, batchIdx int16) (*cql.QueryBuilder, error) {
	dataQb := cql.NewQB().Keyspace(keyspace)
	if err := dataQb.WritePreparedColumn("rowid"); err != nil {
		return nil, err
	}
	if err := dataQb.WritePreparedColumn("batch_idx"); err != nil {
		return nil, err
	}
	if err := dataQb.WritePreparedValue("batch_idx", batchIdx); err != nil {
		return nil, err
	}

	for _, tri := range tableRecordItems {
		if err := dataQb.WritePreparedColumn(tri.FieldName); err != nil {
			return nil, err
		}
	}
	return dataQb, nil
}

func newIdxQueryBuilder(keyspace string) (*cql.QueryBuilder, error) {
	idxQb := cql.NewQB().Keyspace(keyspace)
	if err := idxQb.WritePreparedColumn("key"); err != nil {
		return nil, err
	}
	if err := idxQb.WritePreparedColumn("rowid"); err != nil {
		return nil, err
	}
	return idxQb, nil
}

type PreparedQuery struct {
	Qb    *cql.QueryBuilder
	Query string
}

func (instr *TableInserter) tableNameWithSuffix(tableName string) string {
	return fmt.Sprintf("%s%s", tableName, cql.RunIdSuffix(instr.PCtx.Msg.RunId))
}

// TEST ONLY
// type TestScenario int
// const (
// 	TestDataDoesNotExist TestScenario = iota
// 	TestDataOperationTimedOut
// 	TestDataSerious
// 	TestDataNotApplied
// 	TestIdxDoesNotExist
// 	TestIdxOperationTimedOut
// 	TestIdxSerious
// 	TestIdxNotAppliedSamePresentFirstRun
// 	TestIdxNotAppliedSamePresentSecondRun
// 	TestIdxNotAppliedDiffPresent
// )
// const CurrentTestScenario TestScenario = TestDataDoesNotExist

var ErrDuplicateRowid = errors.New("duplicate rowid")
var ErrDuplicateKey = errors.New("duplicate key")

func (instr *TableInserter) insertDataRecordWithRowid(logger *l.CapiLogger, tableRecordItems []TableRecordItem, rowid int64, pq *PreparedQuery) error {
	logger.PushF("proc.insertDataRecordWithRowid")
	defer logger.PopF()

	curDataExpBackoffFactor := int64(1)
	var errorToReturn error

	// This is the first item from the channel, initialize prepared query, do this once
	if pq.Qb == nil {
		var err error
		// rowid=?, batch_idx=123, field1=?, field2=?
		pq.Qb, err = newDataQueryBuilder(instr.PCtx.Msg.DataKeyspace, tableRecordItems, instr.PCtx.Msg.BatchIdx)
		if err != nil {
			return fmt.Errorf("cannot create insert data query builder: %s", err.Error())
		}
		// insert into table.name_run_id rowid=?, batch_idx=123, field1=?, field2=?
		pq.Query, err = pq.Qb.InsertRunPreparedQuery(
			instr.TableCreator.Name,
			instr.PCtx.Msg.RunId,
			cql.IgnoreIfExists) // INSERT IF NOT EXISTS; if exists,  returned isApplied = false
		if err != nil {
			return fmt.Errorf("cannot prepare insert data query string: %s", err.Error())
		}
	}

	// field1=123, field2=456
	for _, tri := range tableRecordItems {
		if err := pq.Qb.WritePreparedValue(tri.FieldName, tri.Value); err != nil {
			return fmt.Errorf("cannot write prepared value for %s: %s", tri.FieldName, err.Error())
		}
	}

	for retryCount := 0; retryCount < instr.MaxDbProblemRetries; retryCount++ {
		// rowid=111
		if err := pq.Qb.WritePreparedValue("rowid", rowid); err != nil {
			return fmt.Errorf("cannot write rowid to prepared query: %s", err.Error())
		}

		// returns "field1":123, "field2":456
		preparedDataQueryParams, err := pq.Qb.InsertRunParams()
		if err != nil {
			return fmt.Errorf("cannot generate insert params for prepared query %s: %s", pq.Query, err.Error())
		}

		existingDataRow := map[string]any{}
		var isApplied bool
		isApplied, err = instr.PCtx.CqlSession.Query(pq.Query, preparedDataQueryParams...).MapScanCAS(existingDataRow)

		// TEST ONLY (comment out pq.Qb.InsertRunParams() and instr.PCtx.CqlSession.Query() above)
		// var err error
		// if dataRetryCount == 0 {
		// 	instr.DoesNotExistPause = 100      // speed things up for testing
		// 	instr.OperationTimedOutPause = 100 // speed things up for testing
		// 	if CurrentTestScenario == TestDataDoesNotExist {
		// 		// log: will wait for table ... to be created, table retry count 0, got does not exist
		// 		// retry and succeed
		// 		err = fmt.Errorf("does not exist")
		// 	} else if CurrentTestScenario == TestDataOperationTimedOut {
		// 		// log: cluster overloaded (Operation timed out), will wait for ...ms before writing to data table ... again, table retry count 0
		// 		// retry and succeed
		// 		err = fmt.Errorf("Operation timed out")
		// 	} else if CurrentTestScenario == TestDataSerious {
		// 		// UI: some serious error; cannot write to data table
		// 		// give up immediately and report failure
		// 		err = fmt.Errorf("some serious error")
		// 	} else if CurrentTestScenario == TestDataNotApplied {
		// 		// log: duplicate rowid not written [INSERT INTO ...], existing record [...], table retry count 0
		// 		 // retry with new rowid and succeed
		// 		isApplied = false
		// 	}
		// } else {
		// 	preparedDataQueryParams, _ := pq.Qb.InsertRunParams()
		// 	isApplied, err = instr.PCtx.CqlSession.Query(pq.Query, preparedDataQueryParams...).MapScanCAS(existingDataRow)
		// }

		if err == nil {
			if isApplied {
				// Success
				return nil
			}

			// This rowid is already in the db, time to panic. But this is not the end of the world if we are working with distinct_table, so log a warning, not an error
			logger.WarnCtx(instr.PCtx, "duplicate rowid not written [%s], existing record [%v], table retry count %d reached, giving up", pq.Query, existingDataRow, retryCount)
			errorToReturn = fmt.Errorf("cannot write to data table, got rowid duplicate [%s]: %w", pq.Query, ErrDuplicateRowid)
			break
		}
		if strings.Contains(err.Error(), "table ") && strings.Contains(err.Error(), "does not exist") {
			// There is a chance this table is brand new and table schema was not propagated to all Cassandra nodes
			if retryCount >= instr.MaxDbProblemRetries-1 {
				errorToReturn = fmt.Errorf("cannot write to data table %s after %d attempts, apparently, table schema still not propagated to all nodes: %s", instr.tableNameWithSuffix(instr.TableCreator.Name), retryCount+1, err.Error())
				break
			}
			logger.WarnCtx(instr.PCtx, "will wait for table %s to be created, table retry count %d, got %s", instr.tableNameWithSuffix(instr.TableCreator.Name), retryCount, err.Error())
			// TODO: come up with a better waiting strategy (exp backoff, at least)
			time.Sleep(time.Duration(instr.DoesNotExistPauseMillis) * time.Millisecond)
		} else if strings.Contains(err.Error(), "Operation timed out") {
			// The cluster is overloaded, slow down
			if retryCount >= instr.MaxDbProblemRetries-1 {
				errorToReturn = fmt.Errorf("cannot write to data table %s after %d attempts and %dms, still getting timeouts: %s", instr.tableNameWithSuffix(instr.TableCreator.Name), retryCount+1, cql.SumOfExpBackoffDelaysMs(instr.OperationTimedOutPauseMillis, instr.ExpBackoffFactorMultiplier, retryCount), err.Error())
				break
			}
			logger.WarnCtx(instr.PCtx, "cluster overloaded (%s), will wait for %dms before writing to data table %s again, table retry count %d", err.Error(), instr.OperationTimedOutPauseMillis*curDataExpBackoffFactor, instr.tableNameWithSuffix(instr.TableCreator.Name), retryCount)
			time.Sleep(time.Duration(instr.OperationTimedOutPauseMillis*curDataExpBackoffFactor) * time.Millisecond)
			curDataExpBackoffFactor *= instr.ExpBackoffFactorMultiplier
		} else if strings.Contains(err.Error(), "Operation failed - received 0 responses and 1 failures") {
			// Saw this from Amazon Keyspaces, slow down
			if retryCount >= instr.MaxDbProblemRetries-1 {
				errorToReturn = fmt.Errorf("cannot write to data table %s after %d attempts and %dms, still getting zero responses: %s", instr.tableNameWithSuffix(instr.TableCreator.Name), retryCount+1, cql.SumOfExpBackoffDelaysMs(instr.OperationTimedOutPauseMillis, instr.ExpBackoffFactorMultiplier, retryCount), err.Error())
				break
			}
			logger.WarnCtx(instr.PCtx, "cluster returns zero responses (%s), will wait for %dms before writing to data table %s again, table retry count %d", err.Error(), instr.OperationTimedOutPauseMillis*curDataExpBackoffFactor, instr.tableNameWithSuffix(instr.TableCreator.Name), retryCount)
			time.Sleep(time.Duration(instr.OperationTimedOutPauseMillis*curDataExpBackoffFactor) * time.Millisecond)
			curDataExpBackoffFactor *= instr.ExpBackoffFactorMultiplier
		} else if strings.Contains(err.Error(), "The row has exceeded the maximum allowed size") {
			// Saw this from Amazon Keyspaces, give some details
			sb := strings.Builder{}
			sb.WriteString("cannot write to data table, some string lengths exceed max allowed value: ")
			for _, tri := range tableRecordItems {
				switch v := tri.Value.(type) {
				case string:
					sb.WriteString(fmt.Sprintf("%s:%d characters;", tri.FieldName, len(v)))
				default:
					sb.WriteString(fmt.Sprintf("%s:%T;", tri.FieldName, tri.Value))
				}
			}
			errorToReturn = db.WrapDbErrorWithQuery(sb.String(), pq.Query, err)
			break
		} else {
			// Some serious error happened, stop trying this rowid
			errorToReturn = db.WrapDbErrorWithQuery("cannot write to data table", pq.Query, err)
			break
		}
	} // data retry loop

	// Failure
	return errorToReturn
}

func (instr *TableInserter) insertDataRecord(logger *l.CapiLogger, tableRecordItems []TableRecordItem, pq *PreparedQuery, rowidRand *rand.Rand) (int64, error) {
	logger.PushF("proc.insertDataRecord")
	defer logger.PopF()

	curRowid := rowidRand.Int63()
	for retryCount := 0; retryCount < instr.MaxDuplicateRetries; retryCount++ {

		err := instr.insertDataRecordWithRowid(logger, tableRecordItems, curRowid, pq)
		if err == nil {
			return curRowid, nil
		}

		if !errors.Is(err, ErrDuplicateRowid) {
			errorToReturn := fmt.Errorf("cannot insert data record [%s]: %s", pq.Query, err.Error())
			logger.ErrorCtx(instr.PCtx, "%s", errorToReturn.Error())
			return curRowid, errorToReturn
		} else if retryCount < instr.MaxDuplicateRetries-1 {
			rowidRand.Seed(newSeed(instr.MachineHash))
			curRowid = rowidRand.Int63()
		}
		logger.WarnCtx(instr.PCtx, "duplicate rowid not written [%s], rowid retry count %d", pq.Query, retryCount)
	}

	errorToReturn := fmt.Errorf("duplicate rowid not written [%s], giving up after %d rowid retries", pq.Query, instr.MaxDbProblemRetries)
	logger.ErrorCtx(instr.PCtx, "%s", errorToReturn.Error())
	return curRowid, errorToReturn
}

func (instr *TableInserter) insertIdxRecordWithRowid(logger *l.CapiLogger, idxName string, idxUniqueness sc.IdxUniqueness, idxKey string, curRowid int64, pq *PreparedQuery) error {
	logger.PushF("proc.insertIdxRecordWithRowid")
	defer logger.PopF()

	curIdxExpBackoffFactor := int64(1.0)
	var err error

	ifNotExistsFlag := cql.ThrowIfExists
	if idxUniqueness == sc.IdxUnique {
		ifNotExistsFlag = cql.IgnoreIfExists
	}

	// Prepare generic idx insert query, do this once for all indexes and rows
	if pq.Qb == nil {
		// key=?, rowid=?
		pq.Qb, err = newIdxQueryBuilder(instr.PCtx.Msg.DataKeyspace)
		if err != nil {
			return fmt.Errorf("cannot prepare idx builder: %s", err.Error())
		}
	}

	// insert into idx_table.name_run_id key=?, rowid=?
	// idxName is different than on the previous call, update it
	pq.Query, err = pq.Qb.InsertRunPreparedQuery(idxName, instr.PCtx.Msg.RunId, ifNotExistsFlag)
	if err != nil {
		return fmt.Errorf("cannot prepare idx query: %s", err.Error())
	}

	// Provide parameters
	if err := pq.Qb.WritePreparedValue("key", idxKey); err != nil {
		return err
	}
	if err := pq.Qb.WritePreparedValue("rowid", curRowid); err != nil {
		return err
	}

	// returns "key":"aaa", "rowid":123
	preparedIdxQueryParams, err := pq.Qb.InsertRunParams()
	if err != nil {
		return fmt.Errorf("cannot provide idx query params for %s: %s", pq.Query, err.Error())
	}

	var errorToReturn error
	for retryCount := 0; retryCount < instr.MaxDbProblemRetries; retryCount++ {
		existingIdxRow := map[string]any{}
		var isApplied = true

		if idxUniqueness == sc.IdxUnique {
			// Unique idx assumed, check isApplied
			isApplied, err = instr.PCtx.CqlSession.Query(pq.Query, preparedIdxQueryParams...).MapScanCAS(existingIdxRow)
		} else {
			// No uniqueness assumed, just insert
			err = instr.PCtx.CqlSession.Query(pq.Query, preparedIdxQueryParams...).Exec()
		}

		// TEST ONLY (comment out if idxUniqueness == sc.IdxUnique {...} else {...} above)
		// var err error
		// if idxRetryCount == 0 {
		// 	instr.DoesNotExistPause = 100      // speed things up for testing
		// 	instr.OperationTimedOutPause = 100 // speed things up for testing
		// 	if CurrentTestScenario == TestIdxDoesNotExist {
		// 		// log: will wait for idx table ... to be created, table retry count 0, got does not exist
		// 		// retry and succeed
		// 		err = fmt.Errorf("does not exist")
		// 	} else if CurrentTestScenario == TestIdxOperationTimedOut {
		// 		// log: cluster overloaded (Operation timed out), will wait for ...ms before writing to idx table ... again, table retry count 0
		// 		// retry and succeed
		// 		err = fmt.Errorf("Operation timed out")
		// 	} else if CurrentTestScenario == TestIdxSerious {
		// 		// UI: some serious error; cannot insert idx record
		// 		// give up immediately and report failure
		// 		err = fmt.Errorf("some serious error")
		// 	} else if CurrentTestScenario == TestIdxNotAppliedSamePresentFirstRun {
		// 		// UI: cannot write duplicate index key [INSERT INTO ...] and proper rowid with ... on retry 0
		// 		// give up immediately and report failure
		// 		isApplied = false
		// 		existingIdxRow["key"] = idxKey
		// 		existingIdxRow["rowid"] = curRowid
		// 	} else if CurrentTestScenario == TestIdxNotAppliedSamePresentSecondRun {
		// 		// log: duplicate idx record found ... on retry 1 when writing ..., assuming this retry was successful, proceeding as usual
		// 		// consider it a success
		// 		// Simulate first successful attempt:
		// 		if idxUniqueness == sc.IdxUnique {
		// 			isApplied, err = instr.PCtx.CqlSession.Query(pq.Query, preparedIdxQueryParams...).MapScanCAS(existingIdxRow)
		// 		} else {
		// 			err = instr.PCtx.CqlSession.Query(pq.Query, preparedIdxQueryParams...).Exec()
		// 		}
		// 		idxRetryCount = 1 // Pretend it is a second attempt, which makes the key/rowid coincidence legit
		// 		isApplied = false
		// 		existingIdxRow["key"] = idxKey
		// 		existingIdxRow["rowid"] = curRowid
		// 	} else if CurrentTestScenario == TestIdxNotAppliedDiffPresent {
		// 		// UI: cannot write duplicate index key ... with ... on retry 0, existing record [...], rowid is different
		// 		// give up immediately and report failure
		// 		isApplied = false
		// 		existingIdxRow["key"] = idxKey
		// 		existingIdxRow["rowid"] = curRowid + 1
		// 	}
		// } else {
		// 	if idxUniqueness == sc.IdxUnique {
		// 		isApplied, err = instr.PCtx.CqlSession.Query(pq.Query, preparedIdxQueryParams...).MapScanCAS(existingIdxRow)
		// 	} else {
		// 		err = instr.PCtx.CqlSession.Query(pq.Query, preparedIdxQueryParams...).Exec()
		// 	}
		// }

		if err == nil {
			if !isApplied {
				if existingIdxRow["key"] != idxKey || existingIdxRow["rowid"] != curRowid {
					// We screwed up, a record with this key and different rowid is already there, report everything we can
					errorToReturn = fmt.Errorf("cannot write duplicate index key [%s] with %s,%d on retry %d, existing record [%v], rowid is different, throwing werror %w", pq.Query, idxKey, curRowid, retryCount, existingIdxRow, ErrDuplicateKey)
					break
				}
				if retryCount == 0 {
					// This is the first attempt, and the record we neeed is already there (key and rowid are the same). Doesn't sound right.
					errorToReturn = fmt.Errorf("cannot write duplicate index key [%s] and proper rowid with %s,%d on retry %d, existing record [%v], assuming it was some other writer, throwing error %w", pq.Query, idxKey, curRowid, retryCount, existingIdxRow, ErrDuplicateKey)
					break
				}
				// Assuming Cassandra managed to insert the record on the previous attempt but returned an error
				logger.WarnCtx(instr.PCtx, "duplicate idx record found (%s) in idx %s on retry %d when writing (%d,'%s'), assuming this retry was successful, proceeding as usual", idxName, existingIdxRow, retryCount, curRowid, idxKey)
			}
			// Success or not - we are done
			return nil
		}

		if strings.Contains(err.Error(), "does not exist") {
			// There is a chance this table is brand new and table schema was not propagated to all Cassandra nodes
			if retryCount >= instr.MaxDbProblemRetries-1 {
				errorToReturn = fmt.Errorf("cannot write to idx table %s after %d attempts, apparently, table schema still not propagated to all nodes: %s", instr.tableNameWithSuffix(idxName), retryCount+1, err.Error())
				break
			}
			logger.WarnCtx(instr.PCtx, "will wait for idx table %s to be created, table retry count %d, got %s", instr.tableNameWithSuffix(idxName), retryCount, err.Error())
			// TODO: come up with a better waiting strategy (exp backoff, at least)
			time.Sleep(time.Duration(instr.DoesNotExistPauseMillis) * time.Millisecond)
		} else if strings.Contains(err.Error(), "Operation timed out") {
			// The cluster is overloaded, slow down
			if retryCount >= instr.MaxDbProblemRetries-1 {
				errorToReturn = fmt.Errorf("cannot write to idx table %s after %d attempts and %dms, still getting timeout: %s", instr.tableNameWithSuffix(idxName), retryCount+1, cql.SumOfExpBackoffDelaysMs(instr.OperationTimedOutPauseMillis, instr.ExpBackoffFactorMultiplier, retryCount), err.Error())
				break
			}
			logger.WarnCtx(instr.PCtx, "cluster overloaded (%s), will wait for %dms before writing to idx table %s again, table retry count %d", err.Error(), instr.OperationTimedOutPauseMillis*curIdxExpBackoffFactor, instr.tableNameWithSuffix(idxName), retryCount)
			time.Sleep(time.Duration(instr.OperationTimedOutPauseMillis*curIdxExpBackoffFactor) * time.Millisecond)
			curIdxExpBackoffFactor *= instr.ExpBackoffFactorMultiplier
		} else if strings.Contains(err.Error(), "Operation failed - received 0 responses and 1 failures") {
			// Saw this from Amazon Keyspaces, slow down
			if retryCount >= instr.MaxDbProblemRetries-1 {
				errorToReturn = fmt.Errorf("cannot write to idx table %s after %d attempts and %dms, still getting zero responses: %s", instr.tableNameWithSuffix(idxName), retryCount+1, cql.SumOfExpBackoffDelaysMs(instr.OperationTimedOutPauseMillis, instr.ExpBackoffFactorMultiplier, retryCount), err.Error())
				break
			}
			logger.WarnCtx(instr.PCtx, "cluster returns zero responses (%s), will wait for %dms before writing to idx table %s again, table retry count %d", err.Error(), instr.OperationTimedOutPauseMillis*curIdxExpBackoffFactor, instr.tableNameWithSuffix(idxName), retryCount)
			time.Sleep(time.Duration(instr.OperationTimedOutPauseMillis*curIdxExpBackoffFactor) * time.Millisecond)
			curIdxExpBackoffFactor *= instr.ExpBackoffFactorMultiplier
		} else {
			// Some serious error happened, stop trying this idx record
			errorToReturn = db.WrapDbErrorWithQuery("cannot write to idx table", pq.Query, err)
			break
		}
	} // idx retry loop

	// Failure
	return errorToReturn
}

func (instr *TableInserter) insertDistinctIdxAndDataRecords(logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext, tableRecordItems []TableRecordItem, idxName string, keyValue string, pdq *PreparedQuery, piq *PreparedQuery, rowidRand *rand.Rand) (int64, error) {
	logger.PushF("proc.insertDistinctIdxAndDataRecords")
	defer logger.PopF()

	curRowid := rowidRand.Int63()
	for retryCount := 0; retryCount < instr.MaxDuplicateRetries; retryCount++ {
		errInsertIdx := instr.insertIdxRecordWithRowid(logger, idxName, sc.IdxUnique, keyValue, curRowid, piq)
		if errInsertIdx == nil {
			errInsertData := instr.insertDataRecordWithRowid(logger, tableRecordItems, curRowid, pdq)
			if errInsertData == nil {
				return curRowid, nil
			}

			if !errors.Is(errInsertData, ErrDuplicateRowid) {
				return curRowid, errInsertData
			}
			// Delete inserted idx record before trying another rowid
			errDelete := deleteIdxRecordByKey(pCtx, idxName, []string{keyValue})
			if errDelete != nil {
				return curRowid, errDelete
			}
			logger.InfoCtx(pCtx, "cannot insert duplicate rowid on %d attempt: key %s, rowid %d", retryCount, keyValue, curRowid)
		} else if errors.Is(errInsertIdx, ErrDuplicateKey) {
			// ErrDuplicateKey is ok, this means we already have a distinct record, nothing to do here
			logger.DebugCtx(pCtx, "already have a distinct record, nothing to do here: key %s, rowid %d", keyValue, curRowid)
			return curRowid, nil
		} else if retryCount < instr.MaxDuplicateRetries-1 {
			rowidRand.Seed(newSeed(instr.MachineHash))
		} else {
			// Some serious error
			return curRowid, errInsertIdx
		}
	} // retry loop

	errorToReport := fmt.Errorf("cannot insert distinct idx/data records after %d attempts", instr.MaxDuplicateRetries)
	logger.ErrorCtx(pCtx, "%s", errorToReport.Error())
	return curRowid, errorToReport
}

func (instr *TableInserter) insertIdxRecordsForIndexes(logger *l.CapiLogger, writeItem *WriteChannelItem, idxNameToSkip string, newRowid int64, piq *PreparedQuery) error {
	for _, ikmi := range writeItem.IndexKeyItems {
		if idxNameToSkip != "" && ikmi.IdxName == idxNameToSkip {
			continue
		}
		idxDef, ok := instr.TableCreator.Indexes[ikmi.IdxName]
		if !ok {
			return fmt.Errorf("dev error, inserter cannot find index %s in %v", ikmi.IdxName, instr.TableCreator.Indexes)
		}
		if err := instr.insertIdxRecordWithRowid(logger, ikmi.IdxName, idxDef.Uniqueness, ikmi.KeyValue, newRowid, piq); err != nil {
			return fmt.Errorf("cannot insert idx record to %s: %s", ikmi.IdxName, err.Error())
		}
	}

	return nil
}

func (instr *TableInserter) tableInserterWorker(logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext) {
	logger.PushF("proc.tableInserterWorker")
	defer logger.PopF()

	// Each writer thread has its own rand, so we do not have to critsec it.
	// Assuming machine hashes are different for all daemon machines!
	rowidRand := rand.New(rand.NewSource(newSeed(instr.MachineHash)))

	pdq := PreparedQuery{}
	piq := PreparedQuery{}

	logger.DebugCtx(pCtx, "started reading from RecordsIn")

	handledRecordCount := 0

	// For each record in instr.RecordsIn, we MUST produce one item in instr.RecordWrittenStatuses
	for writeItem := range instr.RecordsIn {
		handledRecordCount++
		var errorToReport error
		switch instr.DataIdxSeqMode {
		case DataIdxSeqModeDataFirst:
			var newRowid int64
			newRowid, errorToReport = instr.insertDataRecord(logger, writeItem.TableRecordItems, &pdq, rowidRand)
			if errorToReport == nil {
				// Index tables
				err := instr.insertIdxRecordsForIndexes(logger, &writeItem, "", newRowid, &piq)
				if err != nil {
					errorToReport = fmt.Errorf("cannot insert index records for DataFirst: %s", err.Error())
				}
			}
		case DataIdxSeqModeDistinctIdxFirst:
			// Assuming there is only one index def here, and it's unique
			distinctIdxName, _, err := instr.TableCreator.GetSingleUniqueIndexDef()
			if err != nil {
				errorToReport = fmt.Errorf("unsupported configuration error: %s", err.Error())
			} else {
				var newRowid int64
				distinctIdxKeyVal, err := writeItem.findIndexKeyValByName(distinctIdxName)
				if err != nil {
					errorToReport = fmt.Errorf("unexpectedly cannot find key value for distinct index: %s", err.Error())
				} else {
					newRowid, errorToReport = instr.insertDistinctIdxAndDataRecords(logger, pCtx, writeItem.TableRecordItems, distinctIdxName, distinctIdxKeyVal, &pdq, &piq, rowidRand)
					if errorToReport == nil {
						// Create records for other indexes if any (they all must be non-unique)
						err := instr.insertIdxRecordsForIndexes(logger, &writeItem, distinctIdxName, newRowid, &piq)
						if err != nil {
							errorToReport = fmt.Errorf("cannot insert index records for IdxFirst: %s", err.Error())
						}
					}
				}
			}
		default:
			errorToReport = fmt.Errorf("unsupported instr.DataIdxSeqMode %d", instr.DataIdxSeqMode)
		}

		instr.RecordWrittenStatuses <- errorToReport
	} // items loop

	logger.DebugCtx(pCtx, "done reading from RecordsIn, this writer worker handled %d records from instr.RecordsIn", handledRecordCount)

	// Decrease busy worker count
	instr.WorkerWaitGroup.Done()
}
