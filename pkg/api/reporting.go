package api

import (
	"github.com/capillariesio/capillaries/pkg/gocqlshims"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/wfdb"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
)

// Used by Toolbelt (get_run_history command) to retrieve run status history for a keyspace (used in integration tests)
func GetRunHistory(logger *l.CapiLogger, cqlSession gocqlshims.Session, keyspace string) ([]*wfmodel.RunHistoryEvent, error) {
	logger.PushF("api.GetRunHistory")
	defer logger.PopF()
	rows, err := wfdb.GetRunHistory(logger, cqlSession, keyspace, nil)
	if err != nil {
		return nil, err
	}
	return wfmodel.RunHistoryRowsToEvents(rows)
}

// Used by Webapi and Toolbelt (get_node_history, get_run_status_diagram commands) to retrieve each node status history for multiple runs (used by WebUI main screen and in integration tests)
func GetNodeHistoryForRuns(logger *l.CapiLogger, cqlSession gocqlshims.Session, keyspace string, runIds []int16) ([]*wfmodel.NodeHistoryEvent, error) {
	logger.PushF("api.GetNodeHistoryForRuns")
	defer logger.PopF()
	rows, err := wfdb.GetNodeHistoryForRuns(logger, cqlSession, keyspace, runIds, nil)
	if err != nil {
		return nil, err
	}
	return wfmodel.NodeHistoryRowsToEvents(rows)
}

// Used in Toolbelt (get_batch_history command)
func GetBatchHistoryForRunAndNode(logger *l.CapiLogger, cqlSession gocqlshims.Session, keyspace string, runId int16, nodeName string) ([]*wfmodel.BatchHistoryEvent, error) {
	logger.PushF("api.GetRunNodeBatchHistory")
	defer logger.PopF()
	rows, err := wfdb.GetBatchHistoryForRunAndNode(logger, cqlSession, keyspace, runId, nodeName)
	if err != nil {
		return nil, err
	}
	return wfmodel.BatchHistoryRowsToEvents(rows)
}
