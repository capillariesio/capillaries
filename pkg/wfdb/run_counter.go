package wfdb

import (
	"errors"
	"fmt"
	"math"

	"github.com/capillariesio/capillaries/pkg/cql"
	"github.com/capillariesio/capillaries/pkg/db"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
	"github.com/capillariesio/gocqlmem/gocqlshims"
)

func GetNextRunCounter(logger *l.CapiLogger, cqlSession gocqlshims.Session, keyspace string) (int16, error) {
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
			return 0, fmt.Errorf("cannot get run counter, wrong number of rows: %s", q)
		}

		// last_run is technically Cassandra INT, so it is supposed to be Go int32, but I am getting Go int from gocql. Play safe.
		var lastRunId int64
		switch typedLastRunId := rows[0]["last_run"].(type) {
		case int:
			lastRunId = int64(typedLastRunId)
		case int8:
			lastRunId = int64(typedLastRunId)
		case int16:
			lastRunId = int64(typedLastRunId)
		case int32:
			lastRunId = int64(typedLastRunId)
		case int64:
			lastRunId = typedLastRunId
		default:
			return 0, fmt.Errorf("cannot get run counter from [%v]: %s", rows[0], q)
		}

		// lastRunId, ok := rows[0]["last_run"].(int)
		// if !ok {
		// 	return 0, fmt.Errorf("cannot get run counter from [%v], %s: expected int, got %T", rows[0], q, rows[0]["last_run"])
		// }

		// Try incrementing
		newRunId := lastRunId + 1
		// Run ids are int16 everywhere (RunIdSuffix, message RunId, etc.). Once the counter
		// passes 32767 the int16(newRunId) cast below wraps to a negative value and eventually
		// reuses ids, silently overlapping different runs' per-run data/idx tables. Fail loudly
		// instead of wrapping.
		if newRunId > math.MaxInt16 {
			return 0, fmt.Errorf("run counter exhausted for keyspace %s: reached the maximum of %d runs", keyspace, math.MaxInt16)
		}
		// The If(...) makes this an atomic compare-and-set (LWT). It is the entire anti-collision
		// mechanism: drop it for a plain UPDATE and two concurrent callers can read the same
		// last_run and both claim the same new run id. On a lost race isApplied is false and we retry.
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
	return 0, errors.New("cannot increment run counter, too many attempts")
}
