package proc

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/capillariesio/capillaries/pkg/cql"
	"github.com/capillariesio/capillaries/pkg/ctx"
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
	MinWriterRate   int
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
	seedCounter++
	return (time.Now().Unix() << 32) + time.Now().UnixMilli() + seedCounter
}

func newTableInserter(envConfig *env.EnvConfig, logger *l.Logger, pCtx *ctx.MessageProcessingContext, tableCreator *sc.TableCreatorDef, batchSize int) *TableInserter {

	return &TableInserter{
		PCtx:          pCtx,
		TableCreator:  tableCreator,
		BatchSize:     batchSize,
		ErrorsOut:     make(chan error, batchSize),
		RowidRand:     rand.New(rand.NewSource(newSeed())),
		NumWorkers:    envConfig.Cassandra.WriterWorkers,
		MinWriterRate: envConfig.Cassandra.MinInserterRate,
		RecordsInOpen: false,
		//Logger:        logger,
	}
}

func CreateDataTableCql(keyspace string, runId int16, tableCreator *sc.TableCreatorDef) string {
	qb := cql.QueryBuilder{}
	qb.ColumnDef("rowid", sc.FieldTypeInt)
	qb.ColumnDef("batch_idx", sc.FieldTypeInt)
	for fieldName, fieldDef := range tableCreator.Fields {
		qb.ColumnDef(fieldName, fieldDef.Type)
	}
	return qb.PartitionKey("rowid").Keyspace(keyspace).CreateRun(tableCreator.Name, runId, cql.IgnoreIfExists)
}

func CreateIdxTableCql(keyspace string, runId int16, idxName string, idxDef *sc.IdxDef) string {
	qb := cql.QueryBuilder{}
	qb.Keyspace(keyspace).
		ColumnDef("key", sc.FieldTypeString).
		ColumnDef("rowid", sc.FieldTypeInt)
		//ColumnDef("batch_idx", sc.FieldTypeInt)
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
// 		return cql.WrapDbErrorWithQuery("cannot create data table", q, err)
// 	}

// 	for idxName, idxDef := range instr.TableCreator.Indexes {
// 		q := CreateIdxTableCql(instr.PCtx.BatchInfo.DataKeyspace, instr.PCtx.BatchInfo.RunId, idxName, idxDef)
// 		if err := instr.PCtx.CqlSession.Query(q).Exec(); err != nil {
// 			return cql.WrapDbErrorWithQuery("cannot create idx table", q, err)
// 		}
// 	}
// 	return nil
// }

func (instr *TableInserter) startWorkers(logger *l.Logger, pCtx *ctx.MessageProcessingContext) error {
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

func (instr *TableInserter) waitForWorkers(logger *l.Logger, pCtx *ctx.MessageProcessingContext) error {
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
			//logger.DebugCtx(pCtx, "got result for sent record %d out of %d from instr.ErrorsOut, %d errors so far", i, instr.RecordsSent, errCount)

			inserterRate := float64(i+1) / time.Now().Sub(startTime).Seconds()
			// If it falls below min rate, it does not make sense to continue
			if i > 5 && inserterRate < float64(instr.MinWriterRate) {
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
	} else {
		return nil
	}
}

func (instr *TableInserter) waitForWorkersAndCloseErrorsOut(logger *l.Logger, pCtx *ctx.MessageProcessingContext) {
	logger.PushF("proc.waitForWorkersAndClose/TableInserter")
	defer logger.PopF()

	// Make sure no workers are running, so they do not hit closed ErrorsOut
	instr.waitForWorkers(logger, pCtx)
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

func (instr *TableInserter) tableInserterWorker(logger *l.Logger, pCtx *ctx.MessageProcessingContext) {
	logger.PushF("proc.tableInserterWorker")
	defer logger.PopF()

	logger.DebugCtx(pCtx, "writer started reading from RecordsIn")
	dataTableName := instr.TableCreator.Name + cql.RunIdSuffix(instr.PCtx.BatchInfo.RunId)

	handledRecordCount := 0
	for writeItem := range instr.RecordsIn {
		handledRecordCount++
		maxDataRetries := 5
		curDataExpBackoffFactor := 1
		var errorToReport error

		instr.RandMutex.Lock()
		(*writeItem.TableRecord)["rowid"] = instr.RowidRand.Int63()
		instr.RandMutex.Unlock()

		qb := cql.QueryBuilder{}
		qb.Write("batch_idx", instr.PCtx.BatchInfo.BatchIdx)
		for fieldName, fieldValue := range *writeItem.TableRecord {
			qb.Write(fieldName, fieldValue)
		}
		q := qb.Keyspace(instr.PCtx.BatchInfo.DataKeyspace).
			InsertRun(instr.TableCreator.Name, instr.PCtx.BatchInfo.RunId, cql.IgnoreIfExists) // INSERT IF NOT EXISTS; if exists,  returned isApplied = false

		for dataRetryCount := 0; dataRetryCount < maxDataRetries; dataRetryCount++ {

			existingDataRow := map[string]interface{}{}
			isApplied, err := instr.PCtx.CqlSession.Query(q).MapScanCAS(existingDataRow)

			if err == nil {
				if isApplied {
					// Success
					break
				} else {
					if dataRetryCount < maxDataRetries-1 {
						// Retry now with a new rowid
						logger.InfoCtx(instr.PCtx, "duplicate rowid not written [%s], existing record [%v], retry count %d", q, existingDataRow, dataRetryCount)
						instr.RandMutex.Lock()
						instr.RowidRand = rand.New(rand.NewSource(newSeed()))
						(*writeItem.TableRecord)["rowid"] = instr.RowidRand.Int63()
						instr.RandMutex.Unlock()

						newQb := cql.QueryBuilder{}
						newQb.Write("batch_idx", instr.PCtx.BatchInfo.BatchIdx)
						for fieldName, fieldValue := range *writeItem.TableRecord {
							newQb.Write(fieldName, fieldValue)
						}
						q = newQb.Keyspace(instr.PCtx.BatchInfo.DataKeyspace).
							InsertRun(instr.TableCreator.Name, instr.PCtx.BatchInfo.RunId, cql.IgnoreIfExists) // INSERT IF NOT EXISTS; if exists,  returned isApplied = false
					} else {
						// No more retries
						logger.ErrorCtx(instr.PCtx, "duplicate rowid not written [%s], existing record [%v], retry count %d reached, giving up", q, existingDataRow, dataRetryCount)
						errorToReport = fmt.Errorf("cannot write to data table after multiple attempts, keep getting rowid duplicates [%s]", q)
						break
					}
				}
			} else {
				if strings.Contains(err.Error(), "does not exist") {
					// There is a chance this table is brand new and table schema was not propagated to all Cassandra nodes
					if dataRetryCount < maxDataRetries-1 {
						logger.WarnCtx(instr.PCtx, "will wait for table %s to be created, retry count %d, got %s", dataTableName, dataRetryCount, err.Error())
						// TODO: come up with a better waiting strategy (exp backoff, at least)
						time.Sleep(5 * time.Second)
					} else {
						errorToReport = fmt.Errorf("cannot write to data table %s after %d attempts, apparently, table schema still not propagated to all nodes: %s", dataTableName, dataRetryCount+1, err.Error())
						break
					}
				} else if strings.Contains(err.Error(), "Operation timed out") {
					// The cluster is overloaded, slow down
					if dataRetryCount < maxDataRetries-1 {
						logger.WarnCtx(instr.PCtx, "cluster overloaded (%s), will wait for %dms before writing to data table %s again, retry count %d", err.Error(), 10*curDataExpBackoffFactor, dataTableName, dataRetryCount)
						time.Sleep(time.Duration(10*curDataExpBackoffFactor) * time.Millisecond)
						curDataExpBackoffFactor *= 2
					} else {
						errorToReport = fmt.Errorf("cannot write to data table %s after %d attempts, still getting timeouts: %s", dataTableName, dataRetryCount+1, err.Error())
						break
					}
				} else {
					// Some serious error happened, stop trying this rowid
					errorToReport = cql.WrapDbErrorWithQuery("cannot write to data table", q, err)
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
				qb := cql.QueryBuilder{}
				qb.Write("key", writeItem.IndexKeyMap[idxName])
				qb.Write("rowid", (*writeItem.TableRecord)["rowid"])
				q := qb.Keyspace(instr.PCtx.BatchInfo.DataKeyspace).
					InsertRun(idxName, instr.PCtx.BatchInfo.RunId, ifNotExistsFlag)

				for idxRetryCount := 0; idxRetryCount < maxIdxRetries; idxRetryCount++ {
					existingIdxRow := map[string]interface{}{}
					var isApplied = true
					var err error
					if idxDef.Uniqueness == sc.IdxUnique {
						// Unique idx assumed, check isApplied
						isApplied, err = instr.PCtx.CqlSession.Query(q).MapScanCAS(existingIdxRow)
					} else {
						// No uniqueness assumed, just insert
						err = instr.PCtx.CqlSession.Query(q).Exec()
					}

					if err == nil {
						if !isApplied {
							// If attempt number > 0, there is a chance that Cassandra managed to insert the record on the previous attempt but returned an error
							if idxRetryCount > 0 && existingIdxRow["key"] == writeItem.IndexKeyMap[idxName] && existingIdxRow["rowid"] == (*writeItem.TableRecord)["rowid"] {
								// Cassandra screwed up, but we know how to handle it, the record is there, just log a warning
								logger.WarnCtx(instr.PCtx, "duplicate idx record found (%s) in idx %s on retry %d when writing (%d,'%s'), assuming this retry was successful, proceeding as usual", idxName, existingIdxRow, idxRetryCount, (*writeItem.TableRecord)["rowid"], writeItem.IndexKeyMap[idxName])
							} else {
								// We screwed up, report everything we can
								errorToReport = fmt.Errorf("cannot write duplicate index key [%s] on retry %d, existing record [%v]", q, idxRetryCount, existingIdxRow)
							}
						}
						// Success or not - we are done
						break
					} else {
						if strings.Contains(err.Error(), "does not exist") {
							// There is a chance this table is brand new and table schema was not propagated to all Cassandra nodes
							if idxRetryCount < maxIdxRetries-1 {
								logger.WarnCtx(instr.PCtx, "will wait for idx table %s to be created, retry count %d, got %s", idxTableName, idxRetryCount, err.Error())
								// TODO: come up with a better waiting strategy (exp backoff, at least)
								time.Sleep(5 * time.Second)
							} else {
								errorToReport = fmt.Errorf("cannot write to idx table %s after %d attempts, apparently, table schema still not propagated to all nodes: %s", idxTableName, idxRetryCount+1, err.Error())
								break
							}
						} else if strings.Contains(err.Error(), "Operation timed out") {
							// The cluster is overloaded, slow down
							if idxRetryCount < maxIdxRetries-1 {
								logger.WarnCtx(instr.PCtx, "cluster overloaded (%s), will wait for %dms before writing to idx table %s again, retry count %d", err.Error(), 10*curIdxExpBackoffFactor, idxTableName, idxRetryCount)
								time.Sleep(time.Duration(10*curIdxExpBackoffFactor) * time.Millisecond)
								curIdxExpBackoffFactor *= 2
							} else {
								errorToReport = fmt.Errorf("cannot write to idx table %s after %d attempts, still getting timeout: %s", idxTableName, idxRetryCount+1, err.Error())
								break
							}
						} else {
							// Some serious error happened, stop trying this idx record
							errorToReport = cql.WrapDbErrorWithQuery("cannot write to idx table", q, err)
							break
						}
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
