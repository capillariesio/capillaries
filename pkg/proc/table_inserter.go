package proc

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/capillariesio/capillaries/pkg/cql"
	"github.com/capillariesio/capillaries/pkg/ctx"
	"github.com/capillariesio/capillaries/pkg/db"
	"github.com/capillariesio/capillaries/pkg/env"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/sc"
)

type TableInserter struct {
	PCtx            *ctx.MessageProcessingContext
	TableCreator    *sc.TableCreatorDef
	BatchSize       int
	RecordsIn       chan WriteChannelItem // Channel to pass records from the main function like RunCreateTableForBatch, usig add(), to TableInserter
	ErrorsOut       chan error
	RowidRand       *rand.Rand
	RandMutex       sync.Mutex
	NumWorkers      int
	MinInserterRate int
	WorkerWaitGroup sync.WaitGroup
	RecordsSent     int // Records sent to RecordsIn
	// TODO: the only reason we have this is because we decided to end handlers
	// with "defer instr.waitForWorkersAndCloseErrorsOut(logger, pCtx)" - not the cleanest way, get rid of this bool thingy.
	// That defer is convenient because there are so many early returns.
	RecordsInOpen          bool
	DoesNotExistPause      float32
	OperationTimedOutPause float32
}

type WriteChannelItem struct {
	TableRecord *TableRecord
	IndexKeyMap map[string]string
}

var seedCounter = int64(0)

func newSeed() int64 {
	seedCounter += 3333
	return (time.Now().Unix() << 32) + time.Now().UnixMilli() + seedCounter
}

func newTableInserter(envConfig *env.EnvConfig, pCtx *ctx.MessageProcessingContext, tableCreator *sc.TableCreatorDef, batchSize int) *TableInserter {

	return &TableInserter{
		PCtx:                   pCtx,
		TableCreator:           tableCreator,
		BatchSize:              batchSize,
		ErrorsOut:              make(chan error, batchSize),
		RowidRand:              rand.New(rand.NewSource(newSeed())),
		NumWorkers:             envConfig.Cassandra.WriterWorkers,
		MinInserterRate:        envConfig.Cassandra.MinInserterRate,
		RecordsInOpen:          false,
		DoesNotExistPause:      5.0,  // sec
		OperationTimedOutPause: 10.0, // sec
	}
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

func (instr *TableInserter) startWorkers(logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext) error {
	logger.PushF("proc.startWorkers/TableInserter")
	defer logger.PopF()

	instr.RecordsIn = make(chan WriteChannelItem, instr.BatchSize)
	logger.DebugCtx(pCtx, "startWorkers created RecordsIn,now launching %d writers...", instr.NumWorkers)
	instr.RecordsInOpen = true

	for w := 0; w < instr.NumWorkers; w++ {
		newLogger, err := l.NewLoggerFromLogger(logger)
		if err != nil {
			return err
		}
		// Increase busy worker count
		instr.WorkerWaitGroup.Add(1)
		go instr.tableInserterWorker(newLogger, pCtx)
	}
	return nil
}

func (instr *TableInserter) waitForWorkers(logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext) error {
	logger.PushF("proc.waitForWorkers/TableInserter")
	defer logger.PopF()

	logger.DebugCtx(pCtx, "started reading RecordsSent=%d from instr.ErrorsOut", instr.RecordsSent)

	errors := make([]string, 0)
	if instr.RecordsSent > 0 {
		errCount := 0
		startTime := time.Now()
		// 1. It's crucial that the number of errors to receive eventually should match instr.RecordsSent
		// 2. We do not need an extra select/timeout here - we are guaranteed to receive something in instr.ErrorsOut because of cassndra read timeouts (5-15s or so)
		for i := 0; i < instr.RecordsSent; i++ {
			err := <-instr.ErrorsOut
			if err != nil {
				errors = append(errors, err.Error())
				errCount++
			}

			inserterRate := float64(i+1) / time.Since(startTime).Seconds()
			// If it falls below min rate, it does not make sense to continue
			if i > 5 && inserterRate < float64(instr.MinInserterRate) {
				logger.DebugCtx(pCtx, "slow db insertion rate triggered, will stop reading from instr.ErrorsOut")
				errors = append(errors, fmt.Sprintf("table inserter detected slow db insertion rate %.0f records/s, wrote %d records out of %d", inserterRate, i, instr.RecordsSent))
				errCount++
				break
			}
		}
		logger.DebugCtx(pCtx, "done writing RecordsSent=%d from instr.ErrorsOut, %d errors", instr.RecordsSent, errCount)

		// Reset for the next cycle, if it ever happens
		instr.RecordsSent = 0
	} else {
		logger.DebugCtx(pCtx, "no need to waitfor writer results, no records were sent")
	}

	// Close instr.RecordsIn, it will trigger the completion of all writer workers
	if instr.RecordsInOpen {
		logger.DebugCtx(pCtx, "closing RecordsIn")
		close(instr.RecordsIn)
		logger.DebugCtx(pCtx, "closed RecordsIn")
		instr.RecordsInOpen = false
	}

	// Wait for all writer threads to complete, otherwise they will keep writing to instr.ErrorsOut, which can close anytime after we exit this function
	logger.DebugCtx(pCtx, "waiting for writer workers to complete...")
	instr.WorkerWaitGroup.Wait()
	logger.DebugCtx(pCtx, "writer workers are done")

	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, "; "))
	}

	return nil
}

func (instr *TableInserter) waitForWorkersAndCloseErrorsOut(logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext) {
	logger.PushF("proc.waitForWorkersAndClose/TableInserter")
	defer logger.PopF()

	// Make sure no workers are running, so they do not hit closed ErrorsOut
	if err := instr.waitForWorkers(logger, pCtx); err != nil {
		logger.ErrorCtx(pCtx, fmt.Sprintf("error(s) while waiting for workers to complete: %s", err.Error()))
	}

	// Safe to close now
	logger.DebugCtx(pCtx, "closing ErrorsOut")
	close(instr.ErrorsOut)
	logger.DebugCtx(pCtx, "closed ErrorsOut")
}

func (instr *TableInserter) add(tableRecord TableRecord) error {
	indexKeyMap := map[string]string{}
	for idxName, idxDef := range instr.TableCreator.Indexes {
		var err error
		indexKeyMap[idxName], err = sc.BuildKey(tableRecord, idxDef)
		if err != nil {
			return fmt.Errorf("cannot build key for idx %s, table record [%v]: [%s]", idxName, tableRecord, err.Error())
		}
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

func (instr *TableInserter) insertDataRecord(logger *l.CapiLogger, writeItem *WriteChannelItem, pq *PreparedQuery) (int64, error) {
	maxDataRetries := 5
	curDataExpBackoffFactor := float32(1.0)
	curRowid := int64(0)
	var errorToReturn error

	// This is the first item from the channel, initialize prepared query, do this once
	if pq.Qb == nil {
		var err error
		// rowid=?, batch_idx=123, field1=?, field2=?
		pq.Qb, err = newDataQueryBuilder(instr.PCtx.BatchInfo.DataKeyspace, writeItem, instr.PCtx.BatchInfo.BatchIdx)
		if err != nil {
			return curRowid, fmt.Errorf("cannot create insert data query builder: %s", err.Error())
		}
		// insert into table.name_run_id rowid=?, batch_idx=123, field1=?, field2=?
		pq.Query, err = pq.Qb.InsertRunPreparedQuery(
			instr.TableCreator.Name,
			instr.PCtx.BatchInfo.RunId,
			cql.IgnoreIfExists) // INSERT IF NOT EXISTS; if exists,  returned isApplied = false
		if err != nil {
			return curRowid, fmt.Errorf("cannot prepare insert data query string: %s", err.Error())
		}
	}

	instr.RandMutex.Lock()
	curRowid = instr.RowidRand.Int63()
	instr.RandMutex.Unlock()

	// field1=123, field2=456
	for fieldName, fieldValue := range *writeItem.TableRecord {
		if err := pq.Qb.WritePreparedValue(fieldName, fieldValue); err != nil {
			return curRowid, fmt.Errorf("cannot write prepared value for %s: %s", fieldName, err.Error())
		}
	}

	for dataRetryCount := 0; dataRetryCount < maxDataRetries; dataRetryCount++ {
		// rowid=111
		if err := pq.Qb.WritePreparedValue("rowid", curRowid); err != nil {
			return curRowid, fmt.Errorf("cannot write rowid to prepared query: %s", err.Error())
		}

		// returns "field1":123, "field2":456
		preparedDataQueryParams, err := pq.Qb.InsertRunParams()
		if err != nil {
			return curRowid, fmt.Errorf("cannot generate insert params for prepared query %s: %s", pq.Query, err.Error())
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
		// 		// log: will wait for table ... to be created, retry count 0, got does not exist
		// 		// retry and succeed
		// 		err = fmt.Errorf("does not exist")
		// 	} else if CurrentTestScenario == TestDataOperationTimedOut {
		// 		// log: cluster overloaded (Operation timed out), will wait for ...ms before writing to data table ... again, retry count 0
		// 		// retry and succeed
		// 		err = fmt.Errorf("Operation timed out")
		// 	} else if CurrentTestScenario == TestDataSerious {
		// 		// UI: some serious error; cannot write to data table
		// 		// give up immediately and report failure
		// 		err = fmt.Errorf("some serious error")
		// 	} else if CurrentTestScenario == TestDataNotApplied {
		// 		// log: duplicate rowid not written [INSERT INTO ...], existing record [...], retry count 0
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
				return curRowid, nil
			}

			// This rowid is already in the db, retry or give up
			if dataRetryCount >= maxDataRetries-1 {
				// No more retries
				logger.ErrorCtx(instr.PCtx, "duplicate rowid not written [%s], existing record [%v], retry count %d reached, giving up", pq.Query, existingDataRow, dataRetryCount)
				errorToReturn = fmt.Errorf("cannot write to data table after multiple attempts, keep getting rowid duplicates [%s]", pq.Query)
				break
			}

			// Retry now with a new rowid
			logger.InfoCtx(instr.PCtx, "duplicate rowid not written [%s], existing record [%v], retry count %d", pq.Query, existingDataRow, dataRetryCount)
			instr.RandMutex.Lock()
			instr.RowidRand = rand.New(rand.NewSource(newSeed()))
			curRowid = instr.RowidRand.Int63()
			instr.RandMutex.Unlock()
		} else {
			if strings.Contains(err.Error(), "does not exist") {
				// There is a chance this table is brand new and table schema was not propagated to all Cassandra nodes
				if dataRetryCount >= maxDataRetries-1 {
					errorToReturn = fmt.Errorf("cannot write to data table %s after %d attempts, apparently, table schema still not propagated to all nodes: %s", instr.tableNameWithSuffix(instr.TableCreator.Name), dataRetryCount+1, err.Error())
					break
				}
				logger.WarnCtx(instr.PCtx, "will wait for table %s to be created, retry count %d, got %s", instr.tableNameWithSuffix(instr.TableCreator.Name), dataRetryCount, err.Error())
				// TODO: come up with a better waiting strategy (exp backoff, at least)
				time.Sleep(time.Duration(instr.DoesNotExistPause) * time.Second)
			} else if strings.Contains(err.Error(), "Operation timed out") {
				// The cluster is overloaded, slow down
				if dataRetryCount >= maxDataRetries-1 {
					errorToReturn = fmt.Errorf("cannot write to data table %s after %d attempts, still getting timeouts: %s", instr.tableNameWithSuffix(instr.TableCreator.Name), dataRetryCount+1, err.Error())
					break
				}
				logger.WarnCtx(instr.PCtx, "cluster overloaded (%s), will wait for %fms before writing to data table %s again, retry count %d", err.Error(), 10*curDataExpBackoffFactor, instr.tableNameWithSuffix(instr.TableCreator.Name), dataRetryCount)
				time.Sleep(time.Duration(instr.OperationTimedOutPause*curDataExpBackoffFactor) * time.Millisecond)
				curDataExpBackoffFactor *= 2.0
			} else {
				// Some serious error happened, stop trying this rowid
				errorToReturn = db.WrapDbErrorWithQuery("cannot write to data table", pq.Query, err)
				break
			}
		}
	} // data retry loop

	// Failure
	return curRowid, errorToReturn
}

func (instr *TableInserter) insertIdxRecord(logger *l.CapiLogger, idxName string, idxUniqueness sc.IdxUniqueness, idxKey string, curRowid int64, pq *PreparedQuery) error {
	maxIdxRetries := 5
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
	for idxRetryCount := 0; idxRetryCount < maxIdxRetries; idxRetryCount++ {
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
		// 		// log: will wait for idx table ... to be created, retry count 0, got does not exist
		// 		// retry and succeed
		// 		err = fmt.Errorf("does not exist")
		// 	} else if CurrentTestScenario == TestIdxOperationTimedOut {
		// 		// log: cluster overloaded (Operation timed out), will wait for ...ms before writing to idx table ... again, retry count 0
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
		// 		// Simulate first successfull attempt:
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
				if existingIdxRow["key"] == idxKey && existingIdxRow["rowid"] == curRowid {
					if idxRetryCount > 0 {
						// Assuming Cassandra managed to insert the record on the previous attempt but returned an error
						logger.WarnCtx(instr.PCtx, "duplicate idx record found (%s) in idx %s on retry %d when writing (%d,'%s'), assuming this retry was successful, proceeding as usual", idxName, existingIdxRow, idxRetryCount, curRowid, idxKey)
					} else {
						// This is the first attempt, and the record we neeed is already there. Doesn't sound right
						errorToReturn = fmt.Errorf("cannot write duplicate index key [%s] and proper rowid with %s,%d on retry %d, existing record [%v], assuming it was some other writer, throwing error", pq.Query, idxKey, curRowid, idxRetryCount, existingIdxRow)
						break
					}
				} else {
					// We screwed up, a record with this key and different rowid is already there, report everything we can
					errorToReturn = fmt.Errorf("cannot write duplicate index key [%s] with %s,%d on retry %d, existing record [%v], rowid is different", pq.Query, idxKey, curRowid, idxRetryCount, existingIdxRow)
					break
				}
			}
			// Success or not - we are done
			return nil
		}

		if strings.Contains(err.Error(), "does not exist") {
			// There is a chance this table is brand new and table schema was not propagated to all Cassandra nodes
			if idxRetryCount >= maxIdxRetries-1 {
				errorToReturn = fmt.Errorf("cannot write to idx table %s after %d attempts, apparently, table schema still not propagated to all nodes: %s", instr.tableNameWithSuffix(idxName), idxRetryCount+1, err.Error())
				break
			}
			logger.WarnCtx(instr.PCtx, "will wait for idx table %s to be created, retry count %d, got %s", instr.tableNameWithSuffix(idxName), idxRetryCount, err.Error())
			// TODO: come up with a better waiting strategy (exp backoff, at least)
			time.Sleep(time.Duration(instr.DoesNotExistPause) * time.Second)
		} else if strings.Contains(err.Error(), "Operation timed out") {
			// The cluster is overloaded, slow down
			if idxRetryCount >= maxIdxRetries-1 {
				errorToReturn = fmt.Errorf("cannot write to idx table %s after %d attempts, still getting timeout: %s", instr.tableNameWithSuffix(idxName), idxRetryCount+1, err.Error())
				break
			}
			logger.WarnCtx(instr.PCtx, "cluster overloaded (%s), will wait for %fms before writing to idx table %s again, retry count %d", err.Error(), 10*curIdxExpBackoffFactor, instr.tableNameWithSuffix(idxName), idxRetryCount)
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

func (instr *TableInserter) tableInserterWorker(logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext) {
	logger.PushF("proc.tableInserterWorker")
	defer logger.PopF()

	logger.DebugCtx(pCtx, "writer started reading from RecordsIn")

	pdq := PreparedQuery{}
	piq := PreparedQuery{}

	handledRecordCount := 0
	for writeItem := range instr.RecordsIn {
		handledRecordCount++
		var errorToReport error

		curRowid, errorToReport := instr.insertDataRecord(logger, &writeItem, &pdq)
		if errorToReport == nil {
			// Index tables
			for idxName, idxDef := range instr.TableCreator.Indexes {
				if err := instr.insertIdxRecord(logger, idxName, idxDef.Uniqueness, writeItem.IndexKeyMap[idxName], curRowid, &piq); err != nil {
					errorToReport = fmt.Errorf("cannot insert idx record: %s", err.Error())
					break
				}
			} // idx loop
		}
		instr.ErrorsOut <- errorToReport
	} // items loop
	logger.DebugCtx(pCtx, "done reading from RecordsIn, this writer worker handled %d records from instr.RecordsIn", handledRecordCount)
	// Decrease busy worker count
	instr.WorkerWaitGroup.Done()
}
