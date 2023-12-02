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
	RecordsInOpen bool
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
		PCtx:            pCtx,
		TableCreator:    tableCreator,
		BatchSize:       batchSize,
		ErrorsOut:       make(chan error, batchSize),
		RowidRand:       rand.New(rand.NewSource(newSeed())),
		NumWorkers:      envConfig.Cassandra.WriterWorkers,
		MinInserterRate: envConfig.Cassandra.MinInserterRate,
		RecordsInOpen:   false,
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

// Obsolete: now we create all run-specific tables in api.StartRun
//
// func (instr *TableInserter) verifyTablesExist() error {
// 	q := CreateDataTableCql(instr.PCtx.BatchInfo.DataKeyspace, instr.PCtx.BatchInfo.RunId, instr.TableCreator)
// 	if err := instr.PCtx.CqlSession.Query(q).Exec(); err != nil {
// 		return db.WrapDbErrorWithQuery("cannot create data table", q, err)
// 	}

// 	for idxName, idxDef := range instr.TableCreator.Indexes {
// 		q := CreateIdxTableCql(instr.PCtx.BatchInfo.DataKeyspace, instr.PCtx.BatchInfo.RunId, idxName, idxDef)
// 		if err := instr.PCtx.CqlSession.Query(q).Exec(); err != nil {
// 			return db.WrapDbErrorWithQuery("cannot create idx table", q, err)
// 		}
// 	}
// 	return nil
// }

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

func (instr *TableInserter) tableInserterWorker(logger *l.CapiLogger, pCtx *ctx.MessageProcessingContext) {
	logger.PushF("proc.tableInserterWorker")
	defer logger.PopF()

	logger.DebugCtx(pCtx, "writer started reading from RecordsIn")
	dataTableName := instr.TableCreator.Name + cql.RunIdSuffix(instr.PCtx.BatchInfo.RunId)

	var dataQb *cql.QueryBuilder
	var preparedDataQueryErr error
	var preparedDataQuery string

	handledRecordCount := 0
	for writeItem := range instr.RecordsIn {
		handledRecordCount++
		maxDataRetries := 5
		curDataExpBackoffFactor := 1
		var errorToReport error

		if preparedDataQueryErr != nil {
			instr.ErrorsOut <- fmt.Errorf("cannot prepare data query: %s", preparedDataQueryErr)
			continue
		} else if dataQb == nil {
			dataQb = cql.NewQB()
			if err := dataQb.WritePreparedColumn("rowid"); err != nil {
				instr.ErrorsOut <- errorToReport
				continue // next insert
			}
			if err := dataQb.WritePreparedColumn("batch_idx"); err != nil {
				instr.ErrorsOut <- errorToReport
				continue // next insert
			}
			if err := dataQb.WritePreparedValue("batch_idx", instr.PCtx.BatchInfo.BatchIdx); err != nil {
				instr.ErrorsOut <- errorToReport
				continue // next insert
			}

			for fieldName := range *writeItem.TableRecord {
				if err := dataQb.WritePreparedColumn(fieldName); err != nil {
					errorToReport = fmt.Errorf("cannot prepare data query: %s", err)
					break
				}
			}
			if errorToReport != nil {
				instr.ErrorsOut <- errorToReport
				continue // next insert
			}

			var err error
			preparedDataQuery, err = dataQb.Keyspace(instr.PCtx.BatchInfo.DataKeyspace).
				InsertRunPreparedQuery(instr.TableCreator.Name, instr.PCtx.BatchInfo.RunId, cql.IgnoreIfExists) // INSERT IF NOT EXISTS; if exists,  returned isApplied = false
			if err != nil {
				instr.ErrorsOut <- fmt.Errorf("cannot prepare data query: %s", err)
				continue // next insert
			}
		}

		instr.RandMutex.Lock()
		(*writeItem.TableRecord)["rowid"] = instr.RowidRand.Int63()
		instr.RandMutex.Unlock()

		for fieldName, fieldValue := range *writeItem.TableRecord {
			if err := dataQb.WritePreparedValue(fieldName, fieldValue); err != nil {
				instr.ErrorsOut <- fmt.Errorf("cannot write prepared value for %s: %s", fieldName, err.Error())
				continue // next insert
			}
		}
		preparedDataQueryParams, err := dataQb.InsertRunParams()
		if err != nil {
			instr.ErrorsOut <- fmt.Errorf("cannot provide insert params for prepared query %s: %s", preparedDataQuery, err.Error())
			continue // next insert
		}

		for dataRetryCount := 0; dataRetryCount < maxDataRetries; dataRetryCount++ {

			existingDataRow := map[string]any{}
			isApplied, err := instr.PCtx.CqlSession.Query(preparedDataQuery, preparedDataQueryParams...).MapScanCAS(existingDataRow)

			if err == nil {
				if isApplied {
					// Success
					break
				}

				// This rowidw as already there, retry or give up
				if dataRetryCount >= maxDataRetries-1 {
					// No more retries
					logger.ErrorCtx(instr.PCtx, "duplicate rowid not written [%s], existing record [%v], retry count %d reached, giving up", preparedDataQuery, existingDataRow, dataRetryCount)
					errorToReport = fmt.Errorf("cannot write to data table after multiple attempts, keep getting rowid duplicates [%s]", preparedDataQuery)
					break
				}

				// Retry now with a new rowid
				logger.InfoCtx(instr.PCtx, "duplicate rowid not written [%s], existing record [%v], retry count %d", preparedDataQuery, existingDataRow, dataRetryCount)
				instr.RandMutex.Lock()
				instr.RowidRand = rand.New(rand.NewSource(newSeed()))
				(*writeItem.TableRecord)["rowid"] = instr.RowidRand.Int63()
				instr.RandMutex.Unlock()

				// Set new rowid and re-build query params array (shouldn't throw errors this time)
				if err := dataQb.WritePreparedValue("rowid", (*writeItem.TableRecord)["rowid"]); err != nil {
					errorToReport = fmt.Errorf("cannot prepared value to rowid: %s", err.Error())
					break
				}

				// Will retry (if retry count allows)
				preparedDataQueryParams, _ = dataQb.InsertRunParams()
			} else {
				if strings.Contains(err.Error(), "does not exist") {
					// There is a chance this table is brand new and table schema was not propagated to all Cassandra nodes
					if dataRetryCount >= maxDataRetries-1 {
						errorToReport = fmt.Errorf("cannot write to data table %s after %d attempts, apparently, table schema still not propagated to all nodes: %s", dataTableName, dataRetryCount+1, err.Error())
						break
					}

					logger.WarnCtx(instr.PCtx, "will wait for table %s to be created, retry count %d, got %s", dataTableName, dataRetryCount, err.Error())
					// TODO: come up with a better waiting strategy (exp backoff, at least)
					time.Sleep(5 * time.Second)
				} else if strings.Contains(err.Error(), "Operation timed out") {
					// The cluster is overloaded, slow down
					if dataRetryCount >= maxDataRetries-1 {
						errorToReport = fmt.Errorf("cannot write to data table %s after %d attempts, still getting timeouts: %s", dataTableName, dataRetryCount+1, err.Error())
						break
					}
					logger.WarnCtx(instr.PCtx, "cluster overloaded (%s), will wait for %dms before writing to data table %s again, retry count %d", err.Error(), 10*curDataExpBackoffFactor, dataTableName, dataRetryCount)
					time.Sleep(time.Duration(10*curDataExpBackoffFactor) * time.Millisecond)
					curDataExpBackoffFactor *= 2
				} else {
					// Some serious error happened, stop trying this rowid
					errorToReport = db.WrapDbErrorWithQuery("cannot write to data table", preparedDataQuery, err)
					break
				}
			}
		} // data retry loop

		if errorToReport == nil {
			// Index tables
			for idxName, idxDef := range instr.TableCreator.Indexes {

				maxIdxRetries := 5
				idxTableName := idxName + cql.RunIdSuffix(instr.PCtx.BatchInfo.RunId)
				curIdxExpBackoffFactor := 1

				ifNotExistsFlag := cql.ThrowIfExists
				if idxDef.Uniqueness == sc.IdxUnique {
					ifNotExistsFlag = cql.IgnoreIfExists
				}

				idxQb := cql.NewQB()
				if err := idxQb.WritePreparedColumn("key"); err != nil {
					errorToReport = err
					break
				}
				if err := idxQb.WritePreparedValue("key", writeItem.IndexKeyMap[idxName]); err != nil {
					errorToReport = err
					break
				}
				if err := idxQb.WritePreparedColumn("rowid"); err != nil {
					errorToReport = err
					break
				}
				if err := idxQb.WritePreparedValue("rowid", (*writeItem.TableRecord)["rowid"]); err != nil {
					errorToReport = err
					break
				}

				preparedIdxQuery, err := idxQb.Keyspace(instr.PCtx.BatchInfo.DataKeyspace).InsertRunPreparedQuery(idxName, instr.PCtx.BatchInfo.RunId, ifNotExistsFlag)
				if err != nil {
					errorToReport = fmt.Errorf("cannot prepare idx query: %s", err.Error())
					break
				}
				preparedIdxQueryParams, err := idxQb.InsertRunParams()
				if err != nil {
					errorToReport = fmt.Errorf("cannot provide idx query params for %s: %s", preparedIdxQuery, err.Error())
					break
				}

				for idxRetryCount := 0; idxRetryCount < maxIdxRetries; idxRetryCount++ {
					existingIdxRow := map[string]any{}
					var isApplied = true
					var err error
					if idxDef.Uniqueness == sc.IdxUnique {
						// Unique idx assumed, check isApplied
						isApplied, err = instr.PCtx.CqlSession.Query(preparedIdxQuery, preparedIdxQueryParams...).MapScanCAS(existingIdxRow)
					} else {
						// No uniqueness assumed, just insert
						err = instr.PCtx.CqlSession.Query(preparedIdxQuery, preparedIdxQueryParams...).Exec()
					}

					if err == nil {
						if !isApplied {
							// If attempt number > 0, there is a chance that Cassandra managed to insert the record on the previous attempt but returned an error
							if idxRetryCount > 0 && existingIdxRow["key"] == writeItem.IndexKeyMap[idxName] && existingIdxRow["rowid"] == (*writeItem.TableRecord)["rowid"] {
								// Cassandra screwed up, but we know how to handle it, the record is there, just log a warning
								logger.WarnCtx(instr.PCtx, "duplicate idx record found (%s) in idx %s on retry %d when writing (%d,'%s'), assuming this retry was successful, proceeding as usual", idxName, existingIdxRow, idxRetryCount, (*writeItem.TableRecord)["rowid"], writeItem.IndexKeyMap[idxName])
							} else {
								// We screwed up, report everything we can
								errorToReport = fmt.Errorf("cannot write duplicate index key [%s] on retry %d, existing record [%v]", preparedDataQuery, idxRetryCount, existingIdxRow)
							}
						}
						// Success or not - we are done
						break
					}
					if strings.Contains(err.Error(), "does not exist") {
						// There is a chance this table is brand new and table schema was not propagated to all Cassandra nodes
						if idxRetryCount >= maxIdxRetries-1 {
							errorToReport = fmt.Errorf("cannot write to idx table %s after %d attempts, apparently, table schema still not propagated to all nodes: %s", idxTableName, idxRetryCount+1, err.Error())
							break
						}
						logger.WarnCtx(instr.PCtx, "will wait for idx table %s to be created, retry count %d, got %s", idxTableName, idxRetryCount, err.Error())
						// TODO: come up with a better waiting strategy (exp backoff, at least)
						time.Sleep(5 * time.Second)
					} else if strings.Contains(err.Error(), "Operation timed out") {
						// The cluster is overloaded, slow down
						if idxRetryCount >= maxIdxRetries-1 {
							errorToReport = fmt.Errorf("cannot write to idx table %s after %d attempts, still getting timeout: %s", idxTableName, idxRetryCount+1, err.Error())
							break
						}
						logger.WarnCtx(instr.PCtx, "cluster overloaded (%s), will wait for %dms before writing to idx table %s again, retry count %d", err.Error(), 10*curIdxExpBackoffFactor, idxTableName, idxRetryCount)
						time.Sleep(time.Duration(10*curIdxExpBackoffFactor) * time.Millisecond)
						curIdxExpBackoffFactor *= 2
					} else {
						// Some serious error happened, stop trying this idx record
						errorToReport = db.WrapDbErrorWithQuery("cannot write to idx table", preparedDataQuery, err)
						break
					}
				} // idx retry loop
			} // idx loop
		}
		// logger.DebugCtx(pCtx, "writer wrote")
		instr.ErrorsOut <- errorToReport
	}
	logger.DebugCtx(pCtx, "done reading from RecordsIn, this writer worker handled %d records from instr.RecordsIn", handledRecordCount)
	// Decrease busy worker count
	instr.WorkerWaitGroup.Done()
}
