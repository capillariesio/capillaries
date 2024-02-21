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
	DataIdxSeqModeDataFirst DataIdxSeqModeType = iota
	DataIdxSeqModeDistinctIdxFirst
)

type TableInserter struct {
	PCtx                       *ctx.MessageProcessingContext
	TableCreator               *sc.TableCreatorDef
	RecordsIn                  chan WriteChannelItem // Channel to pass records from the main function like RunCreateTableForBatch, usig add(), to TableInserter
	RecordWrittenStatuses      chan error
	RecordWrittenStatusesMutex sync.Mutex
	MachineHash                int64
	NumWorkers                 int
	MinInserterRate            int
	WorkerWaitGroup            sync.WaitGroup
	RecordsSent                int // Records sent to RecordsIn
	RecordsProcessed           int // Number of items received in RecordWrittenStatuses
	DoesNotExistPause          float32
	OperationTimedOutPause     float32
	DataIdxSeqMode             DataIdxSeqModeType
}

type WriteChannelItem struct {
	TableRecord *TableRecord
	IndexKeyMap map[string]string
}

var seedCounter = int64(0)

func newSeed(hash int64) int64 {
	return hash + atomic.AddInt64(&seedCounter, 1)
}

func createInserterAndStartWorkers(logger *l.CapiLogger, envConfig *env.EnvConfig, pCtx *ctx.MessageProcessingContext, tableCreator *sc.TableCreatorDef, channelSize int, dataIdxSeqMode DataIdxSeqModeType, stringForHash string) (*TableInserter, error) {
	logger.PushF("proc.createInserterAndStartWorkers/TableInserter")
	defer logger.PopF()

	stringHash := fnv.New64a()
	if _, err := stringHash.Write([]byte(stringForHash)); err != nil {
		return nil, err
	}

	instr := &TableInserter{
		PCtx:                   pCtx,
		TableCreator:           tableCreator,
		RecordsIn:              make(chan WriteChannelItem, channelSize), // Capacity should match RecordWrittenStatuses
		RecordWrittenStatuses:  make(chan error, channelSize),            // Capacity should match RecordsIn
		MachineHash:            int64(stringHash.Sum64()),
		NumWorkers:             envConfig.Cassandra.WriterWorkers,
		MinInserterRate:        envConfig.Cassandra.MinInserterRate,
		RecordsSent:            0,    // Total number of records added to RecordsIn
		RecordsProcessed:       0,    // Total number of records read from RecordsOut
		DoesNotExistPause:      5.0,  // sec
		OperationTimedOutPause: 10.0, // sec
		DataIdxSeqMode:         dataIdxSeqMode,
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
	return qb.PartitionKey("rowid").Keyspace(keyspace).CreateRun(tableCreator.Name, runId, cql.IgnoreIfExists)
}

func CreateIdxTableCql(keyspace string, runId int16, idxName string, idxDef *sc.IdxDef) string {
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
	return qb.CreateRun(idxName, runId, cql.IgnoreIfExists)
}

func (instr *TableInserter) letWorkersDrainRecordWrittenStatuses(logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext) error {
	logger.PushF("proc.letWorkersDrainRecordWrittenStatuses/TableInserter")
	defer logger.PopF()

	drainedRecordCount := 0
	errorsFound := make([]string, 0)
	errCount := 0
	startTime := time.Now()
	logger.DebugCtx(pCtx, "started draining at RecordsSent=%d from instr.RecordWrittenStatuses", instr.RecordsSent)

	// 1. It's crucial that the number of errors to receive eventually should match instr.RecordsSent
	// 2. We do not need an extra select/timeout here - we are guaranteed to receive something in instr.RecordWrittenStatuses because of cassandra read timeouts (5-15s or so)
	for instr.RecordsSent > instr.RecordsProcessed {
		err := <-instr.RecordWrittenStatuses
		instr.RecordsProcessed++
		drainedRecordCount++
		if err != nil {
			errorsFound = append(errorsFound, err.Error())
			errCount++
		}

		// If it falls below min rate, it does not make sense to continue
		inserterRate := float64(drainedRecordCount) / time.Since(startTime).Seconds()
		if drainedRecordCount > 5 && inserterRate < float64(instr.MinInserterRate) {
			logger.DebugCtx(pCtx, "slow db insertion rate triggered, will stop reading from instr.RecordWrittenStatuses")
			errorsFound = append(errorsFound, fmt.Sprintf("table inserter detected slow db insertion rate %.0f records/s, wrote %d records out of %d", inserterRate, drainedRecordCount, instr.RecordsSent))
			errCount++
			break
		}
	}
	logger.DebugCtx(pCtx, "done draining %d records at RecordsSent=%d from instr.RecordWrittenStatuses, %d errors", drainedRecordCount, instr.RecordsSent, errCount)

	if len(errorsFound) > 0 {
		return fmt.Errorf(strings.Join(errorsFound, "; "))
	}

	return nil
}

func (instr *TableInserter) letWorkersDrainRecordWrittenStatusesAndCloseInserter(logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext) {
	logger.PushF("proc.letWorkersDrainRecordWrittenStatusesAndCloseInserter/TableInserter")
	defer logger.PopF()

	// If anything was sent at all - drain
	if instr.RecordsSent > 0 {
		if err := instr.letWorkersDrainRecordWrittenStatuses(logger, pCtx); err != nil {
			logger.ErrorCtx(pCtx, fmt.Sprintf("error(s) while waiting for workers to drain RecordsIn: %s", err.Error()))
		}
	}

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
}

func (instr *TableInserter) buildIndexKeys(tableRecord TableRecord) (map[string]string, error) {
	indexKeyMap := map[string]string{}
	for idxName, idxDef := range instr.TableCreator.Indexes {
		var err error
		indexKeyMap[idxName], err = sc.BuildKey(tableRecord, idxDef)
		if err != nil {
			return nil, fmt.Errorf("cannot build key for idx %s, table record [%v]: [%s]", idxName, tableRecord, err.Error())
		}
	}

	return indexKeyMap, nil
}

func (instr *TableInserter) add(logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext, tableRecord TableRecord, indexKeyMap map[string]string) error {
	for len(instr.RecordsIn) == cap(instr.RecordsIn) {
		logger.DebugCtx(pCtx, "RecordsIn cap %d reached, waiting for workers to drain RecordsIn...", cap(instr.RecordsIn))
		time.Sleep(100 * time.Millisecond)
	}

	instr.RecordsSent++
	instr.RecordsIn <- WriteChannelItem{TableRecord: &tableRecord, IndexKeyMap: indexKeyMap}

	return nil
}

func newDataQueryBuilder(keyspace string, writeItem *WriteChannelItem, batchIdx int16) (*cql.QueryBuilder, error) {
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

	for fieldName := range *writeItem.TableRecord {
		if err := dataQb.WritePreparedColumn(fieldName); err != nil {
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
	return fmt.Sprintf("%s%s", tableName, cql.RunIdSuffix(instr.PCtx.BatchInfo.RunId))
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

func (instr *TableInserter) insertDataRecordWithRowid(logger *l.CapiLogger, writeItem *WriteChannelItem, rowid int64, pq *PreparedQuery) error {
	logger.PushF("proc.insertDataRecordWithRowid")
	defer logger.PopF()

	maxRetries := 5
	curDataExpBackoffFactor := float32(1.0)
	var errorToReturn error

	// This is the first item from the channel, initialize prepared query, do this once
	if pq.Qb == nil {
		var err error
		// rowid=?, batch_idx=123, field1=?, field2=?
		pq.Qb, err = newDataQueryBuilder(instr.PCtx.BatchInfo.DataKeyspace, writeItem, instr.PCtx.BatchInfo.BatchIdx)
		if err != nil {
			return fmt.Errorf("cannot create insert data query builder: %s", err.Error())
		}
		// insert into table.name_run_id rowid=?, batch_idx=123, field1=?, field2=?
		pq.Query, err = pq.Qb.InsertRunPreparedQuery(
			instr.TableCreator.Name,
			instr.PCtx.BatchInfo.RunId,
			cql.IgnoreIfExists) // INSERT IF NOT EXISTS; if exists,  returned isApplied = false
		if err != nil {
			return fmt.Errorf("cannot prepare insert data query string: %s", err.Error())
		}
	}

	// field1=123, field2=456
	for fieldName, fieldValue := range *writeItem.TableRecord {
		if err := pq.Qb.WritePreparedValue(fieldName, fieldValue); err != nil {
			return fmt.Errorf("cannot write prepared value for %s: %s", fieldName, err.Error())
		}
	}

	for retryCount := 0; retryCount < maxRetries; retryCount++ {
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
		// 	instr.DoesNotExistPause = 0.01      // speed things up for testing
		// 	instr.OperationTimedOutPause = 0.01 // speed things up for testing
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
		if strings.Contains(err.Error(), "does not exist") {
			// There is a chance this table is brand new and table schema was not propagated to all Cassandra nodes
			if retryCount >= maxRetries-1 {
				errorToReturn = fmt.Errorf("cannot write to data table %s after %d attempts, apparently, table schema still not propagated to all nodes: %s", instr.tableNameWithSuffix(instr.TableCreator.Name), retryCount+1, err.Error())
				break
			}
			logger.WarnCtx(instr.PCtx, "will wait for table %s to be created, table retry count %d, got %s", instr.tableNameWithSuffix(instr.TableCreator.Name), retryCount, err.Error())
			// TODO: come up with a better waiting strategy (exp backoff, at least)
			time.Sleep(time.Duration(instr.DoesNotExistPause) * time.Second)
		} else if strings.Contains(err.Error(), "Operation timed out") {
			// The cluster is overloaded, slow down
			if retryCount >= maxRetries-1 {
				errorToReturn = fmt.Errorf("cannot write to data table %s after %d attempts, still getting timeouts: %s", instr.tableNameWithSuffix(instr.TableCreator.Name), retryCount+1, err.Error())
				break
			}
			logger.WarnCtx(instr.PCtx, "cluster overloaded (%s), will wait for %fms before writing to data table %s again, table retry count %d", err.Error(), 10*curDataExpBackoffFactor, instr.tableNameWithSuffix(instr.TableCreator.Name), retryCount)
			time.Sleep(time.Duration(instr.OperationTimedOutPause*curDataExpBackoffFactor) * time.Millisecond)
			curDataExpBackoffFactor *= 2.0
		} else {
			// Some serious error happened, stop trying this rowid
			errorToReturn = db.WrapDbErrorWithQuery("cannot write to data table", pq.Query, err)
			break
		}
	} // data retry loop

	// Failure
	return errorToReturn
}

func (instr *TableInserter) insertDataRecord(logger *l.CapiLogger, writeItem *WriteChannelItem, pq *PreparedQuery, rowidRand *rand.Rand) (int64, error) {
	logger.PushF("proc.insertDataRecord")
	defer logger.PopF()

	const maxRetries int = 5
	curRowid := rowidRand.Int63()
	for retryCount := 0; retryCount < maxRetries; retryCount++ {

		err := instr.insertDataRecordWithRowid(logger, writeItem, curRowid, pq)
		if err == nil {
			return curRowid, nil
		}

		if !errors.Is(err, ErrDuplicateRowid) {
			errorToReturn := fmt.Errorf("cannot insert data record [%s]: %s", pq.Query, err.Error())
			logger.ErrorCtx(instr.PCtx, errorToReturn.Error())
			return curRowid, errorToReturn
		} else if retryCount < maxRetries-1 {
			rowidRand.Seed(newSeed(instr.MachineHash))
			curRowid = rowidRand.Int63()
		}
		logger.WarnCtx(instr.PCtx, "duplicate rowid not written [%s], rowid retry count %d", pq.Query, retryCount)
	}

	errorToReturn := fmt.Errorf("duplicate rowid not written [%s], giving up after %d rowid retries", pq.Query, maxRetries)
	logger.ErrorCtx(instr.PCtx, errorToReturn.Error())
	return curRowid, errorToReturn
}

func (instr *TableInserter) insertIdxRecordWithRowid(logger *l.CapiLogger, idxName string, idxUniqueness sc.IdxUniqueness, idxKey string, curRowid int64, pq *PreparedQuery) error {
	logger.PushF("proc.insertIdxRecordWithRowid")
	defer logger.PopF()

	maxRetries := 5
	curIdxExpBackoffFactor := float32(1.0)
	var err error

	ifNotExistsFlag := cql.ThrowIfExists
	if idxUniqueness == sc.IdxUnique {
		ifNotExistsFlag = cql.IgnoreIfExists
	}

	// Prepare generic idx insert query, do this once for all indexes and rows
	if pq.Qb == nil {
		// key=?, rowid=?
		pq.Qb, err = newIdxQueryBuilder(instr.PCtx.BatchInfo.DataKeyspace)
		if err != nil {
			return fmt.Errorf("cannot prepare idx builder: %s", err.Error())
		}
	}

	// insert into idx_table.name_run_id key=?, rowid=?
	// idxName is different than on the previous call, update it
	pq.Query, err = pq.Qb.InsertRunPreparedQuery(idxName, instr.PCtx.BatchInfo.RunId, ifNotExistsFlag)
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
	for retryCount := 0; retryCount < maxRetries; retryCount++ {
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
		// 	instr.DoesNotExistPause = 0.01      // speed things up for testing
		// 	instr.OperationTimedOutPause = 0.01 // speed things up for testing
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
			if retryCount >= maxRetries-1 {
				errorToReturn = fmt.Errorf("cannot write to idx table %s after %d attempts, apparently, table schema still not propagated to all nodes: %s", instr.tableNameWithSuffix(idxName), retryCount+1, err.Error())
				break
			}
			logger.WarnCtx(instr.PCtx, "will wait for idx table %s to be created, table retry count %d, got %s", instr.tableNameWithSuffix(idxName), retryCount, err.Error())
			// TODO: come up with a better waiting strategy (exp backoff, at least)
			time.Sleep(time.Duration(instr.DoesNotExistPause) * time.Second)
		} else if strings.Contains(err.Error(), "Operation timed out") {
			// The cluster is overloaded, slow down
			if retryCount >= maxRetries-1 {
				errorToReturn = fmt.Errorf("cannot write to idx table %s after %d attempts, still getting timeout: %s", instr.tableNameWithSuffix(idxName), retryCount+1, err.Error())
				break
			}
			logger.WarnCtx(instr.PCtx, "cluster overloaded (%s), will wait for %fms before writing to idx table %s again, table retry count %d", err.Error(), 10*curIdxExpBackoffFactor, instr.tableNameWithSuffix(idxName), retryCount)
			time.Sleep(time.Duration(instr.OperationTimedOutPause*curIdxExpBackoffFactor) * time.Millisecond)
			curIdxExpBackoffFactor *= 2.0
		} else {
			// Some serious error happened, stop trying this idx record
			errorToReturn = db.WrapDbErrorWithQuery("cannot write to idx table", pq.Query, err)
			break
		}
	} // idx retry loop

	// Failure
	return errorToReturn
}

func (instr *TableInserter) insertDistinctIdxAndDataRecords(logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext, idxName string, writeItem *WriteChannelItem, pdq *PreparedQuery, piq *PreparedQuery, rowidRand *rand.Rand) (int64, error) {
	logger.PushF("proc.insertDistinctIdxAndDataRecords")
	defer logger.PopF()

	const maxRetries int = 5
	curRowid := rowidRand.Int63()
	for retryCount := 0; retryCount < maxRetries; retryCount++ {
		errInsertIdx := instr.insertIdxRecordWithRowid(logger, idxName, sc.IdxUnique, writeItem.IndexKeyMap[idxName], curRowid, piq)
		if errInsertIdx == nil {
			errInsertData := instr.insertDataRecordWithRowid(logger, writeItem, curRowid, pdq)
			if errInsertData == nil {
				return curRowid, nil
			}

			if !errors.Is(errInsertData, ErrDuplicateRowid) {
				return curRowid, errInsertData
			}
			// Delete inserted idx record before trying another rowid
			errDelete := deleteIdxRecordByKey(logger, pCtx, idxName, []string{writeItem.IndexKeyMap[idxName]})
			if errDelete != nil {
				return curRowid, errDelete
			}
			logger.InfoCtx(pCtx, "cannot insert duplicate rowid on %d attempt: key %s, rowid %d", retryCount, writeItem.IndexKeyMap[idxName], curRowid)
		} else if errors.Is(errInsertIdx, ErrDuplicateKey) {
			// ErrDuplicateKey is ok, this means we already have a distinct record, nothing to do here
			logger.DebugCtx(pCtx, "already have a distinct record, nothing to do here: key %s, rowid %d", writeItem.IndexKeyMap[idxName], curRowid)
			return curRowid, nil
		} else if retryCount < maxRetries-1 {
			rowidRand.Seed(newSeed(instr.MachineHash))
		} else {
			// Some serious error
			return curRowid, errInsertIdx
		}
	} // retry loop

	errorToReport := fmt.Errorf("cannot insert distinct idx/data records after %d attempts", maxRetries)
	logger.ErrorCtx(pCtx, errorToReport.Error())
	return curRowid, errorToReport
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
		if instr.DataIdxSeqMode == DataIdxSeqModeDataFirst {
			var newRowid int64
			newRowid, errorToReport = instr.insertDataRecord(logger, &writeItem, &pdq, rowidRand)
			if errorToReport == nil {
				// Index tables
				for idxName, idxDef := range instr.TableCreator.Indexes {
					if err := instr.insertIdxRecordWithRowid(logger, idxName, idxDef.Uniqueness, writeItem.IndexKeyMap[idxName], newRowid, &piq); err != nil {
						errorToReport = fmt.Errorf("cannot insert idx record: %s", err.Error())
						break
					}
				} // idx loop
			}
		} else if instr.DataIdxSeqMode == DataIdxSeqModeDistinctIdxFirst {
			// Assuming there is only one index def here, and it's unique
			distinctIdxName, _, err := instr.TableCreator.GetSingleUniqueIndexDef()
			if err != nil {
				errorToReport = fmt.Errorf("unsupported configuration error: %s", err.Error())
			} else {
				var newRowid int64
				newRowid, errorToReport = instr.insertDistinctIdxAndDataRecords(logger, pCtx, distinctIdxName, &writeItem, &pdq, &piq, rowidRand)
				if errorToReport == nil {
					// Create records for other indexes if any (they all must be non-unique)
					for idxName, idxDef := range instr.TableCreator.Indexes {
						if idxName == distinctIdxName {
							continue
						}
						if err := instr.insertIdxRecordWithRowid(logger, idxName, idxDef.Uniqueness, writeItem.IndexKeyMap[idxName], newRowid, &piq); err != nil {
							errorToReport = fmt.Errorf("cannot insert idx record: %s", err.Error())
							break
						}
					} // idx loop
				}
			}
		} else {
			errorToReport = fmt.Errorf("unsupported instr.DataIdxSeqMode %d", instr.DataIdxSeqMode)
		}

		instr.RecordWrittenStatusesMutex.Lock()
		for len(instr.RecordWrittenStatuses) == cap(instr.RecordWrittenStatuses) {
			logger.ErrorCtx(pCtx, "cannot write to RecordWrittenStatuses, waiting for letWorkersDrainRecordWrittenStatuses to be called, RecordWrittenStatuses len/cap: %d / %d, RecordsIn len/cap: %d / %d ", len(instr.RecordWrittenStatuses), cap(instr.RecordWrittenStatuses), len(instr.RecordsIn), cap(instr.RecordsIn))
			time.Sleep(100 * time.Millisecond)
		}
		instr.RecordWrittenStatuses <- errorToReport
		instr.RecordWrittenStatusesMutex.Unlock()

	} // items loop

	logger.DebugCtx(pCtx, "done reading from RecordsIn, this writer worker handled %d records from instr.RecordsIn", handledRecordCount)

	// Decrease busy worker count
	instr.WorkerWaitGroup.Done()
}
