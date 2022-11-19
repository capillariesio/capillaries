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
	RecordsIn     chan WriteChannelItem
	ErrorsOut     chan error
	RowidRand     *rand.Rand
	RandMutex     sync.Mutex
	NumWorkers    int
	RecordsSent   int
	RecordsInOpen bool
	//Logger        *l.Logger
}

type WriteChannelItem struct {
	TableRecord *TableRecord
	IndexKeys   *[]string
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

func (instr *TableInserter) startWorkers(logger *l.Logger) error {
	instr.RecordsIn = make(chan WriteChannelItem, instr.BatchSize)
	instr.RecordsInOpen = true

	for w := 0; w < instr.NumWorkers; w++ {
		newLogger, err := l.NewLoggerFromLogger(logger)
		if err != nil {
			return err
		}
		go instr.tableInserterWorker(newLogger)
	}
	return nil
}

func (instr *TableInserter) waitForWorkers() error {

	if instr.RecordsInOpen {
		close(instr.RecordsIn)
		instr.RecordsInOpen = false
	}

	errors := make([]string, 0)
	for i := 0; i < instr.RecordsSent; i++ {
		err := <-instr.ErrorsOut
		if err != nil {
			errors = append(errors, err.Error())
		}
	}

	// Reset for the next cycle, if it ever happens
	instr.RecordsSent = 0

	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, "; "))
	} else {
		return nil
	}
}

func (instr *TableInserter) waitForWorkersAndClose() {
	// Make sure no workers are running, so they do not hit closed ErrorsOut
	instr.waitForWorkers()
	// Safe to close now
	close(instr.ErrorsOut)
}

func (instr *TableInserter) add(tableRecord TableRecord) error {
	indexKeys := make([]string, len(instr.TableCreator.Indexes))
	indexIdx := 0
	for idxName, idxDef := range instr.TableCreator.Indexes {
		var err error
		indexKeys[indexIdx], err = sc.BuildKey(tableRecord, idxDef)
		if err != nil {
			return fmt.Errorf("cannot build key for idx %s, table record [%v]: [%s]", idxName, tableRecord, err.Error())
		}
		indexIdx++
	}

	instr.RecordsSent++
	instr.RecordsIn <- WriteChannelItem{TableRecord: &tableRecord, IndexKeys: &indexKeys}

	return nil
}

func (instr *TableInserter) tableInserterWorker(logger *l.Logger) {
	logger.PushF("proc.tableInserterWorker")
	defer logger.PopF()

	for writeItem := range instr.RecordsIn {
		maxRetries := 3
		var errorToReport error
		for retryCount := 0; retryCount < maxRetries; retryCount++ {

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

			if err != nil {
				// Some serious error happened, stop trying this rowid
				errorToReport = cql.WrapDbErrorWithQuery("cannot write to data table", q, err)
				break
			} else if !isApplied {
				if retryCount < maxRetries-1 {
					// Retry now
					logger.InfoCtx(instr.PCtx, "duplicate rowid not written [%s], existing record [%v], retry count %d", q, existingDataRow, retryCount)
					instr.RandMutex.Lock()
					instr.RowidRand = rand.New(rand.NewSource(newSeed()))
					instr.RandMutex.Unlock()
					continue
				} else {
					// No more retries
					logger.ErrorCtx(instr.PCtx, "duplicate rowid not written [%s], existing record [%v], retry count %d reached, giving up", q, existingDataRow, retryCount)
					errorToReport = fmt.Errorf("cannot write to data table after multiple attempts, keep getting rowid duplicates [%s]", q)
				}
			} else {
				// Index tables
				indexIdx := 0
				for idxName, idxDef := range instr.TableCreator.Indexes {
					ifNotExistsFlag := cql.ThrowIfExists
					if idxDef.Uniqueness == sc.IdxUnique {
						ifNotExistsFlag = cql.IgnoreIfExists
					}
					qb = cql.QueryBuilder{}
					//qb.Write("batch_idx", instr.PCtx.BatchInfo.BatchIdx)
					qb.Write("key", (*writeItem.IndexKeys)[indexIdx])
					qb.Write("rowid", (*writeItem.TableRecord)["rowid"])
					q := qb.Keyspace(instr.PCtx.BatchInfo.DataKeyspace).
						InsertRun(idxName, instr.PCtx.BatchInfo.RunId, ifNotExistsFlag)

					existingIdxRow := map[string]interface{}{}
					var isApplied = true
					if idxDef.Uniqueness == sc.IdxUnique {
						// Unique idx assumed, check isApplied
						isApplied, err = instr.PCtx.CqlSession.Query(q).MapScanCAS(existingIdxRow)
					} else {
						// No uniqueness assumed, just insert
						err = instr.PCtx.CqlSession.Query(q).Exec()
					}

					if err != nil {
						errorToReport = cql.WrapDbErrorWithQuery("cannot write index table", q, err)
						break
					} else if !isApplied {
						errorToReport = fmt.Errorf("cannot write duplicate index key [%s], existing record [%v]", q, existingIdxRow)
						break
					}
					indexIdx++
				}

				// No need to retry
				break
			}
		}
		instr.ErrorsOut <- errorToReport
	}
}
