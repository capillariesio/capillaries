package wfdb

import (
	"fmt"

	"github.com/capillariesio/capillaries/pkg/cql"
	"github.com/capillariesio/capillaries/pkg/db"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
	"github.com/gocql/gocql"
)

func GetNextRunCounter(logger *l.CapiLogger, cqlSession *gocql.Session, keyspace string) (int16, error) {
	logger.PushF("wfdb.GetNextRunCounter")
	defer logger.PopF()

	maxRetries := 100
	for retryCount := 0; retryCount < maxRetries; retryCount++ {

		// Initialize optimistic locking
		q := (&cql.QueryBuilder{}).
			Keyspace(keyspace).
			Select(wfmodel.TableNameRunCounter, []string{"last_run"})
		rows, err := cqlSession.Query(q).Iter().SliceMap()
		if err != nil {
			return 0, db.WrapDbErrorWithQuery("cannot get run counter", q, err)
		}

		if len(rows) != 1 {
			return 0, fmt.Errorf("cannot get run counter, wrong number of rows: %s, %s", q, err.Error())
		}

		lastRunId, ok := rows[0]["last_run"].(int)
		if !ok {
			return 0, fmt.Errorf("cannot get run counter from [%v]: %s, %s", rows[0], q, err.Error())
		}

		// Try incrementing
		newRunId := lastRunId + 1
		q = (&cql.QueryBuilder{}).
			Keyspace(keyspace).
			Write("last_run", newRunId).
			Cond("ks", "=", keyspace).
			If("last_run", "=", lastRunId).
			Update(wfmodel.TableNameRunCounter)
		existingDataRow := map[string]any{}
		isApplied, err := cqlSession.Query(q).MapScanCAS(existingDataRow)

		if err != nil {
			return 0, db.WrapDbErrorWithQuery("cannot increment run counter", q, err)
		} else if isApplied {
			return int16(newRunId), nil
		}

		// Retry
		logger.Info("GetNextRunCounter: retry %d", retryCount)
	}
	return 0, fmt.Errorf("cannot increment run counter, too many attempts")
}
