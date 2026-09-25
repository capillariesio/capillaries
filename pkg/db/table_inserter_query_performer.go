package db

import (
	"github.com/capillariesio/capillaries/pkg/cql"
	"github.com/capillariesio/capillaries/pkg/sc"
	"github.com/capillariesio/gocqlmem/gocqlshims"
)

type TableInserterQueryPerformer interface {
	PerformInsertIdxRecordWithRowid(idxUniqueness sc.IdxUniqueness, gocqlSession gocqlshims.Session, pq *cql.PreparedQuery, preparedIdxQueryParams []any, retryCount int) (map[string]any, int, bool, error)
	PerformInsertDataRecordWithRowid(gocqlSession gocqlshims.Session, pq *cql.PreparedQuery, preparedDataQueryParams []any, retryCount int) (map[string]any, bool, error)
}

func HelperPerformInsertDataRecordWithRowid(gocqlSession gocqlshims.Session, pq *cql.PreparedQuery, preparedDataQueryParams []any) (map[string]any, bool, error) {
	existingDataRow := map[string]any{}
	isApplied, err := gocqlSession.Query(pq.Query, preparedDataQueryParams...).MapScanCAS(existingDataRow)
	return existingDataRow, isApplied, err
}

func HelperPerformInsertIdxRecordWithRowid(idxUniqueness sc.IdxUniqueness, gocqlSession gocqlshims.Session, pq *cql.PreparedQuery, preparedIdxQueryParams []any, retryCount int) (map[string]any, int, bool, error) {
	existingIdxRow := map[string]any{}
	var isApplied bool
	var err error

	if idxUniqueness == sc.IdxUnique {
		// Unique idx assumed, check isApplied
		isApplied, err = gocqlSession.Query(pq.Query, preparedIdxQueryParams...).MapScanCAS(existingIdxRow)
	} else {
		// No uniqueness assumed, just insert
		err = gocqlSession.Query(pq.Query, preparedIdxQueryParams...).Exec()
		isApplied = err == nil
	}
	return existingIdxRow, retryCount, isApplied, err
}

type TableInserterQueryPerformerProduction struct{}

func (qp *TableInserterQueryPerformerProduction) PerformInsertDataRecordWithRowid(gocqlSession gocqlshims.Session, pq *cql.PreparedQuery, preparedDataQueryParams []any, _ int) (map[string]any, bool, error) {
	return HelperPerformInsertDataRecordWithRowid(gocqlSession, pq, preparedDataQueryParams)
}

func (qp *TableInserterQueryPerformerProduction) PerformInsertIdxRecordWithRowid(idxUniqueness sc.IdxUniqueness, gocqlSession gocqlshims.Session, pq *cql.PreparedQuery, preparedIdxQueryParams []any, retryCount int) (map[string]any, int, bool, error) {
	return HelperPerformInsertIdxRecordWithRowid(idxUniqueness, gocqlSession, pq, preparedIdxQueryParams, retryCount)
}
