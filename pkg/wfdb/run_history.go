package wfdb

import (
	"fmt"
	"time"

	"github.com/gocql/gocql"
	"github.com/kleineshertz/capillaries/pkg/cql"
	"github.com/kleineshertz/capillaries/pkg/ctx"
	"github.com/kleineshertz/capillaries/pkg/l"
	"github.com/kleineshertz/capillaries/pkg/wfmodel"
)

func GetCurrentRunStatus(logger *l.Logger, pCtx *ctx.MessageProcessingContext) (wfmodel.RunStatusType, error) {
	logger.PushF("GetCurrentRunStatus")
	defer logger.PopF()

	fields := []string{"ts", "status"}
	qb := cql.QueryBuilder{}
	q := qb.
		Keyspace(pCtx.BatchInfo.DataKeyspace).
		Cond("run_id", "=", pCtx.BatchInfo.RunId).
		Select(wfmodel.TableNameRunHistory, fields)
	rows, err := pCtx.CqlSession.Query(q).Iter().SliceMap()
	if err != nil {
		return wfmodel.RunNone, cql.WrapDbErrorWithQuery(fmt.Sprintf("cannot query run status for %s", pCtx.BatchInfo.FullBatchId()), q, err)
	}

	lastStatus := wfmodel.RunNone
	lastTs := time.Unix(0, 0)
	for _, r := range rows {
		rec, err := wfmodel.NewRunHistoryFromMap(r, fields)
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

func HarvestRunLifespans(logger *l.Logger, pCtx *ctx.MessageProcessingContext, runIds []int16) (wfmodel.RunLifespanMap, error) {
	logger.PushF("HarvestRunStatusesForRunIds")
	defer logger.PopF()

	fields := []string{"ts", "run_id", "status"}
	q := (&cql.QueryBuilder{}).
		Keyspace(pCtx.BatchInfo.DataKeyspace).
		CondInInt16("run_id", runIds).
		Select(wfmodel.TableNameRunHistory, fields)
	rows, err := pCtx.CqlSession.Query(q).Iter().SliceMap()
	if err != nil {
		return nil, cql.WrapDbErrorWithQuery("cannot get run statuses for a list of run ids", q, err)
	}

	runLifespanMap := wfmodel.RunLifespanMap{}
	for _, runId := range runIds {
		runLifespanMap[runId] = &wfmodel.RunLifespan{
			StartTs:      time.Time{},
			LastStatus:   wfmodel.RunNone,
			LastStatusTs: time.Time{}}
	}

	for _, r := range rows {
		rec, err := wfmodel.NewRunHistoryFromMap(r, fields)
		if err != nil {
			return nil, fmt.Errorf("%s, %s", err.Error(), q)
		}

		if rec.Status == wfmodel.RunStart {
			runLifespanMap[rec.RunId].StartTs = rec.Ts
		}

		// Later status wins, Stop always wins
		if rec.Ts.After(runLifespanMap[rec.RunId].LastStatusTs) || rec.Status == wfmodel.RunStop {
			runLifespanMap[rec.RunId].LastStatus = rec.Status
			runLifespanMap[rec.RunId].LastStatusTs = rec.Ts
		}
	}

	logger.DebugCtx(pCtx, "run ids %v, lifespans %s", runIds, runLifespanMap.ToString())
	return runLifespanMap, nil
}

func SetRunStatus(logger *l.Logger, cqlSession *gocql.Session, keyspace string, runId int16, status wfmodel.RunStatusType, comment string, ifNotExistsFlag cql.IfNotExistsType) error {
	logger.PushF("SetRunStatus")
	defer logger.PopF()

	q := (&cql.QueryBuilder{}).
		Keyspace(keyspace).
		WriteForceUnquote("ts", "toTimeStamp(now())").
		Write("run_id", runId).
		Write("status", status).
		Write("comment", comment).
		Insert(wfmodel.TableNameRunHistory, ifNotExistsFlag)
	err := cqlSession.Query(q).Exec()
	if err != nil {
		return cql.WrapDbErrorWithQuery("cannot write run status", q, err)
	}

	logger.Debug("run %d, status %s", runId, status.ToString())
	return nil
}
