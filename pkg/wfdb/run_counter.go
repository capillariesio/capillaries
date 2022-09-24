package wfdb

import (
	"fmt"

	"github.com/gocql/gocql"
	"github.com/kleineshertz/capillaries/pkg/cql"
	"github.com/kleineshertz/capillaries/pkg/l"
	"github.com/kleineshertz/capillaries/pkg/wfmodel"
)

func GetNextRunCounter(logger *l.Logger, cqlSession *gocql.Session, keyspace string) (int16, error) {
	logger.PushF("GetNextRunCounter")
	defer logger.PopF()

	maxRetries := 100
	for retryCount := 0; retryCount < maxRetries; retryCount++ {

		// Initialize optimistic locking
		q := (&cql.QueryBuilder{}).
			Keyspace(keyspace).
			Select(wfmodel.TableNameRunCounter, []string{"last_run"})
		rows, err := cqlSession.Query(q).Iter().SliceMap()
		if err != nil {
			return 0, cql.WrapDbErrorWithQuery("cannot get run counter", q, err)
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
		existingDataRow := map[string]interface{}{}
		isApplied, err := cqlSession.Query(q).MapScanCAS(existingDataRow)

		if err != nil {
			return 0, cql.WrapDbErrorWithQuery("cannot increment run counter", q, err)
		} else if isApplied {
			return int16(newRunId), nil
		}

		// Retry
		logger.Info("GetNextRunCounter: retry %d", retryCount)
	}
	return 0, fmt.Errorf("cannot increment run counter, too many attempts")
}
