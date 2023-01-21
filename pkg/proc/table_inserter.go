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
	PCtx          *ctx.MessageProcessingContext
	TableCreator  *sc.TableCreatorDef
	BatchSize     int
	RecordsIn     chan WriteChannelItem // Channel to pass records from the main function like RunCreateTableForBatch, usig add(), to TableInserter
	ErrorsOut     chan error
	RowidRand     *rand.Rand
	RandMutex     sync.Mutex
	NumWorkers    int
	RecordsSent   int // Records sent to RecordsIn
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

func (instr *TableInserter) verifyTablesExist() error {
	q := CreateDataTableCql(instr.PCtx.BatchInfo.DataKeyspace, instr.PCtx.BatchInfo.RunId, instr.TableCreator)
	if err := instr.PCtx.CqlSession.Query(q).Exec(); err != nil {
		return cql.WrapDbErrorWithQuery("cannot create data table", q, err)
	}

	for idxName, idxDef := range instr.TableCreator.Indexes {
		q := CreateIdxTableCql(instr.PCtx.BatchInfo.DataKeyspace, instr.PCtx.BatchInfo.RunId, idxName, idxDef)
		if err := instr.PCtx.CqlSession.Query(q).Exec(); err != nil {
			return cql.WrapDbErrorWithQuery("cannot create idx table", q, err)
		}
	}
	return nil
}

func (instr *TableInserter) startWorkers(logger *l.Logger, pCtx *ctx.MessageProcessingContext) error {
	instr.RecordsIn = make(chan WriteChannelItem, instr.BatchSize)
	logger.DebugCtx(pCtx, "startWorkers created RecordsIn")
	instr.RecordsInOpen = true

	for w := 0; w < instr.NumWorkers; w++ {
		newLogger, err := l.NewLoggerFromLogger(logger)
		if err != nil {
			return err
		}
		go instr.tableInserterWorker(newLogger, pCtx)
	}
	return nil
}

func (instr *TableInserter) waitForWorkers(logger *l.Logger, pCtx *ctx.MessageProcessingContext) error {
	logger.PushF("proc.waitForWorkers/TableInserter")
	defer logger.PopF()

	if instr.RecordsInOpen {
		logger.DebugCtx(pCtx, "closing RecordsIn")
		close(instr.RecordsIn)
		logger.DebugCtx(pCtx, "closed RecordsIn")
		instr.RecordsInOpen = false
	}

	logger.DebugCtx(pCtx, "started reading from RecordsSent")
	errors := make([]string, 0)
	// It's crucial that the number of errors to receive eventually should match instr.RecordsSent
	for i := 0; i < instr.RecordsSent; i++ {
		err := <-instr.ErrorsOut
		if err != nil {
			errors = append(errors, err.Error())
		}
	}
	logger.DebugCtx(pCtx, "done reading from RecordsSent")

	// Reset for the next cycle, if it ever happens
	instr.RecordsSent = 0

	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, "; "))
	} else {
		return nil
	}
}

func (instr *TableInserter) waitForWorkersAndClose(logger *l.Logger, pCtx *ctx.MessageProcessingContext) {
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

	logger.DebugCtx(pCtx, "started reading from RecordsIn")
	dataTableName := instr.TableCreator.Name + cql.RunIdSuffix(instr.PCtx.BatchInfo.RunId)

	for writeItem := range instr.RecordsIn {
		maxDataRetries := 5
		curDataExpBackoffFactor := 1
		var errorToReport error
		for dataRetryCount := 0; dataRetryCount < maxDataRetries; dataRetryCount++ {
			// Data table
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
						instr.RandMutex.Unlock()
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
						logger.WarnCtx(instr.PCtx, "will wait for table %s to be created, retry count %d", dataTableName, dataRetryCount)
						// TODO: come up with a better waiting strategy (exp backoff, at least)
						time.Sleep(5 * time.Second)
					} else {
						logger.ErrorCtx(instr.PCtx, "table %s still not created, retry count %d reached, giving up", dataTableName, dataRetryCount)
						errorToReport = fmt.Errorf("cannot write to data table after multiple attempts, table %s%d schema still not propagated to all nodes", dataTableName)
						break
					}
				} else if strings.Contains(err.Error(), "Operation timed out") {
					// The cluster is overloaded, slow down
					if dataRetryCount < maxDataRetries-1 {
						logger.WarnCtx(instr.PCtx, "cluster overloaded (%s), will wait for %dms before writing to data table %s again, retry count %d", err.Error(), 10*curDataExpBackoffFactor, dataTableName, dataRetryCount)
						time.Sleep(time.Duration(10*curDataExpBackoffFactor) * time.Millisecond)
						curDataExpBackoffFactor *= 2
					} else {
						logger.ErrorCtx(instr.PCtx, "cluster overloaded (%s), cannot write to data table %s, retry count %d reached, giving up", err.Error(), dataTableName, dataRetryCount)
						errorToReport = fmt.Errorf("cannot write to data table %s after multiple attempts, table schema still not propagated to all nodes", dataTableName)
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

				for idxRetryCount := 0; idxRetryCount < maxIdxRetries; idxRetryCount++ {
					ifNotExistsFlag := cql.ThrowIfExists
					if idxDef.Uniqueness == sc.IdxUnique {
						ifNotExistsFlag = cql.IgnoreIfExists
					}
					qb := cql.QueryBuilder{}
					qb.Write("key", writeItem.IndexKeyMap[idxName])
					qb.Write("rowid", (*writeItem.TableRecord)["rowid"])
					q := qb.Keyspace(instr.PCtx.BatchInfo.DataKeyspace).
						InsertRun(idxName, instr.PCtx.BatchInfo.RunId, ifNotExistsFlag)

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
							// We assume that our previous attempts to write this idx record (if this is not the first retry) did not leave any trace in the database,
							// so finding an existing copy of this record is a problem indeed
							errorToReport = fmt.Errorf("cannot write duplicate index key [%s], existing record [%v]", q, existingIdxRow)
						}
						// Success or not - we are done
						break
					} else {
						if strings.Contains(err.Error(), "does not exist") {
							// There is a chance this table is brand new and table schema was not propagated to all Cassandra nodes
							if idxRetryCount < maxIdxRetries-1 {
								logger.WarnCtx(instr.PCtx, "will wait for idx table %s to be created, retry count %d", idxTableName, idxRetryCount)
								// TODO: come up with a better waiting strategy (exp backoff, at least)
								time.Sleep(5 * time.Second)
							} else {
								logger.ErrorCtx(instr.PCtx, "idx table %s still not created, retry count %d reached, giving up", idxTableName, idxRetryCount)
								errorToReport = fmt.Errorf("cannot write to idx table %s after multiple attempts, table schema still not propagated to all nodes", idxTableName)
								break
							}
						} else if strings.Contains(err.Error(), "Operation timed out") {
							// The cluster is overloaded, slow down
							if idxRetryCount < maxIdxRetries-1 {
								logger.WarnCtx(instr.PCtx, "cluster overloaded (%s), will wait for %dms before writing to idx table %s again, retry count %d", err.Error(), 10*curIdxExpBackoffFactor, idxTableName, idxRetryCount)
								time.Sleep(time.Duration(10*curIdxExpBackoffFactor) * time.Millisecond)
								curIdxExpBackoffFactor *= 2
							} else {
								logger.ErrorCtx(instr.PCtx, "cluster overloaded (%s), cannot write to idx table %s, retry count %d reached, giving up", err.Error(), idxTableName, idxRetryCount)
								errorToReport = fmt.Errorf("cannot write to idx table %s after multiple attempts, table schema still not propagated to all nodes", idxTableName)
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
		instr.ErrorsOut <- errorToReport
	}
	logger.DebugCtx(pCtx, "done reading from RecordsIn")
}
