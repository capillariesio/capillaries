package wfdb

import (
	"fmt"
	"sort"
	"time"

	"github.com/capillariesio/capillaries/pkg/cql"
	"github.com/capillariesio/capillaries/pkg/ctx"
	"github.com/capillariesio/capillaries/pkg/db"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
	"github.com/gocql/gocql"
)

func GetCurrentRunStatus(logger *l.Logger, pCtx *ctx.MessageProcessingContext) (wfmodel.RunStatusType, error) {
	logger.PushF("wfdb.GetCurrentRunStatus")
	defer logger.PopF()

	fields := []string{"ts", "status"}
	qb := cql.QueryBuilder{}
	q := qb.
		Keyspace(pCtx.BatchInfo.DataKeyspace).
		Cond("run_id", "=", pCtx.BatchInfo.RunId).
		Select(wfmodel.TableNameRunHistory, fields)
	rows, err := pCtx.CqlSession.Query(q).Iter().SliceMap()
	if err != nil {
		return wfmodel.RunNone, db.WrapDbErrorWithQuery(fmt.Sprintf("cannot query run status for %s", pCtx.BatchInfo.FullBatchId()), q, err)
	}

	lastStatus := wfmodel.RunNone
	lastTs := time.Unix(0, 0)
	for _, r := range rows {
		rec, err := wfmodel.NewRunHistoryEventFromMap(r, fields)
		if err != nil {
			return wfmodel.RunNone, fmt.Errorf("%s, %s", err.Error(), q)
		}

		if rec.Ts.After(lastTs) {
			lastTs = rec.Ts
			lastStatus = wfmodel.RunStatusType(rec.Status)
		}
	}

	logger.DebugCtx(pCtx, "batch %s, run status %s", pCtx.BatchInfo.FullBatchId(), lastStatus.ToString())
	return lastStatus, nil
}

func HarvestRunLifespans(logger *l.Logger, cqlSession *gocql.Session, keyspace string, runIds []int16) (wfmodel.RunLifespanMap, error) {
	logger.PushF("wfdb.HarvestRunLifespans")
	defer logger.PopF()

	qb := (&cql.QueryBuilder{}).Keyspace(keyspace)
	if len(runIds) > 0 {
		qb.CondInInt16("run_id", runIds)
	}
	q := qb.Select(wfmodel.TableNameRunHistory, wfmodel.RunHistoryEventAllFields())
	rows, err := cqlSession.Query(q).Iter().SliceMap()
	if err != nil {
		return nil, db.WrapDbErrorWithQuery("cannot get run statuses for a list of run ids", q, err)
	}

	events := make([]*wfmodel.RunHistoryEvent, len(rows))

	for idx, r := range rows {
		rec, err := wfmodel.NewRunHistoryEventFromMap(r, wfmodel.RunHistoryEventAllFields())
		if err != nil {
			return nil, fmt.Errorf("%s, %s", err.Error(), q)
		}
		events[idx] = rec
	}

	sort.Slice(events, func(i, j int) bool { return events[i].Ts.Before(events[j].Ts) })

	runLifespanMap := wfmodel.RunLifespanMap{}
	emptyUnix := time.Time{}.Unix()
	for _, e := range events {
		if e.Status == wfmodel.RunStart {
			runLifespanMap[e.RunId] = &wfmodel.RunLifespan{RunId: e.RunId, StartTs: e.Ts, StartComment: e.Comment, FinalStatus: wfmodel.RunStart, CompletedTs: time.Time{}, StoppedTs: time.Time{}}
		} else {
			_, ok := runLifespanMap[e.RunId]
			if !ok {
				return nil, fmt.Errorf("unexpected sequence of run status events: %v, %s", events, q)
			}
			if e.Status == wfmodel.RunComplete && runLifespanMap[e.RunId].CompletedTs.Unix() == emptyUnix {
				runLifespanMap[e.RunId].CompletedTs = e.Ts
				runLifespanMap[e.RunId].CompletedComment = e.Comment
				if runLifespanMap[e.RunId].StoppedTs.Unix() == emptyUnix {
					runLifespanMap[e.RunId].FinalStatus = wfmodel.RunComplete // If it was not stopped so far, consider it complete
				}
			} else if e.Status == wfmodel.RunStop && runLifespanMap[e.RunId].StoppedTs.Unix() == emptyUnix {
				runLifespanMap[e.RunId].StoppedTs = e.Ts
				runLifespanMap[e.RunId].StoppedComment = e.Comment
				runLifespanMap[e.RunId].FinalStatus = wfmodel.RunStop // Stop always wins as final status, it may be sign for dependency checker to declare results invalid (depending on the rules)
			}
		}
	}

	return runLifespanMap, nil
}

func SetRunStatus(logger *l.Logger, cqlSession *gocql.Session, keyspace string, runId int16, status wfmodel.RunStatusType, comment string, ifNotExistsFlag cql.IfNotExistsType) error {
	logger.PushF("wfdb.SetRunStatus")
	defer logger.PopF()

	q := (&cql.QueryBuilder{}).
		Keyspace(keyspace).
		WriteForceUnquote("ts", "toTimeStamp(now())").
		Write("run_id", runId).
		Write("status", status).
		Write("comment", comment).
		InsertUnpreparedQuery(wfmodel.TableNameRunHistory, ifNotExistsFlag)
	err := cqlSession.Query(q).Exec()
	if err != nil {
		return db.WrapDbErrorWithQuery("cannot write run status", q, err)
	}

	logger.Debug("run %d, status %s", runId, status.ToString())
	return nil
}
