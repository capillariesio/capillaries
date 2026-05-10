package wfmodel

import (
	"fmt"
	"slices"
	"time"
)

type NodeBatchStatusType int8

// In priority order
const (
	NodeBatchNone            NodeBatchStatusType = 0
	NodeBatchStart           NodeBatchStatusType = 1
	NodeBatchSuccess         NodeBatchStatusType = 2
	NodeBatchFail            NodeBatchStatusType = 3 // Biz logicerror or data table (not WF) error
	NodeBatchRunStopReceived NodeBatchStatusType = 104
)

func NodeBatchStatusToString(s NodeBatchStatusType) string {
	switch s {
	case NodeBatchNone:
		return "none"
	case NodeBatchStart:
		return "start"
	case NodeBatchSuccess:
		return "success"
	case NodeBatchFail:
		return "fail"
	case NodeBatchRunStopReceived:
		return "stopreceived"
	default:
		return "unknown"
	}
}

const TableNameBatchHistory = "wf_batch_history"

// Object model with tags that allow to create cql CREATE TABLE queries and to print object
type BatchHistoryEvent struct {
	Ts           time.Time           `header:"ts" format:"%-33v" column:"ts" type:"timestamp" json:"ts"`
	RunId        int16               `header:"run_id" format:"%6d" column:"run_id" type:"int" key:"true" json:"run_id"`
	ScriptNode   string              `header:"script_node" format:"%20v" column:"script_node" type:"text" key:"true" json:"script_node"`
	BatchIdx     int16               `header:"bnum" format:"%5v" column:"batch_idx" type:"int" key:"true" json:"batch_idx"`
	BatchesTotal int16               `header:"tbtchs" format:"%6v" column:"batches_total" type:"int" json:"batches_total"`
	Status       NodeBatchStatusType `header:"sts" format:"%3v" column:"status" type:"tinyint" key:"true" json:"status"`
	FirstToken   int64               `header:"ftoken" format:"%21v" column:"first_token" type:"bigint" json:"first_token"`
	LastToken    int64               `header:"ltoken" format:"%21v" column:"last_token" type:"bigint" json:"last_token"`
	Instance     string              `header:"instance" format:"%21v" column:"instance" type:"text" json:"instance"`
	Thread       int64               `header:"thread" format:"%4v" column:"thread" type:"bigint" json:"thread"`
	Comment      string              `header:"comment" format:"%v" column:"comment" type:"text" json:"comment"`
}

func BatchHistoryEventAllFields() []string {
	return []string{"ts", "run_id", "script_node", "batch_idx", "batches_total", "status", "first_token", "last_token", "instance", "thread", "comment"}
}
func NewBatchHistoryEventFromMap(r map[string]any, fields []string) (*BatchHistoryEvent, error) {
	res := &BatchHistoryEvent{}
	for _, fieldName := range fields {
		var err error
		switch fieldName {
		case "ts":
			res.Ts, err = ReadTimeFromRow(fieldName, r)
		case "run_id":
			res.RunId, err = ReadInt16FromRow(fieldName, r)
		case "script_node":
			res.ScriptNode, err = ReadStringFromRow(fieldName, r)
		case "batch_idx":
			res.BatchIdx, err = ReadInt16FromRow(fieldName, r)
		case "batches_total":
			res.BatchesTotal, err = ReadInt16FromRow(fieldName, r)
		case "status":
			res.Status, err = ReadNodeBatchStatusFromRow(fieldName, r)
		case "first_token":
			res.FirstToken, err = ReadInt64FromRow(fieldName, r)
		case "last_token":
			res.LastToken, err = ReadInt64FromRow(fieldName, r)
		case "instance":
			res.Instance, err = ReadStringFromRow(fieldName, r)
		case "thread":
			res.Thread, err = ReadInt64FromRow(fieldName, r)
		case "comment":
			res.Comment, err = ReadStringFromRow(fieldName, r)
		default:
			return nil, fmt.Errorf("unknown %s field %s", TableNameBatchHistory, fieldName)
		}
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

func BatchHistoryRowsToEvents(rows []map[string]any) ([]*BatchHistoryEvent, error) {
	result := make([]*BatchHistoryEvent, len(rows))
	for rowIdx, row := range rows {
		rec, err := NewBatchHistoryEventFromMap(row, BatchHistoryEventAllFields())
		if err != nil {
			return nil, fmt.Errorf("cannot deserialize batch history row %v: %s", row, err.Error())
		}
		result[rowIdx] = rec
	}

	slices.SortFunc(result, func(l, r *BatchHistoryEvent) int {
		switch {
		case l.Ts.Before(r.Ts):
			return -1
		case l.Ts.After(r.Ts):
			return 1
		default:
			return 0
		}
	})
	return result, nil
}

func BatchHistoryRowsToNodeStatus(rows []map[string]any) (NodeBatchStatusType, int, int, error) {
	foundBatchesTotal := int16(-1)
	batchesInProgress := map[int16]struct{}{}

	failFound := false
	stopReceivedFound := false
	for _, r := range rows {
		rec, err := NewBatchHistoryEventFromMap(r, BatchHistoryEventAllFields())
		if err != nil {
			return NodeBatchNone, 0, 0, fmt.Errorf("cannot deserialize batch history row [%v]: %s", r, err.Error())
		}
		if foundBatchesTotal == -1 {
			foundBatchesTotal = rec.BatchesTotal
			for i := int16(0); i < rec.BatchesTotal; i++ {
				batchesInProgress[i] = struct{}{}
			}
		} else if rec.BatchesTotal != foundBatchesTotal {
			return NodeBatchNone, 0, 0, fmt.Errorf("conflicting batches total value, was %d, now %d", foundBatchesTotal, rec.BatchesTotal)
		}

		if rec.BatchIdx >= rec.BatchesTotal || rec.BatchesTotal < 0 || rec.BatchesTotal <= 0 {
			return NodeBatchNone, 0, 0, fmt.Errorf("invalid batch idx/total(%d/%d) when processing [%v]", rec.BatchIdx, rec.BatchesTotal, r)
		}

		if rec.Status == NodeBatchSuccess ||
			rec.Status == NodeBatchFail ||
			rec.Status == NodeBatchRunStopReceived {
			delete(batchesInProgress, rec.BatchIdx)
		}

		switch rec.Status {
		case NodeBatchFail:
			failFound = true
		case NodeBatchRunStopReceived:
			stopReceivedFound = true
		default:
			// Nothing interesting yet
		}
	}

	if len(batchesInProgress) == 0 {
		nodeStatus := NodeBatchSuccess
		if stopReceivedFound {
			nodeStatus = NodeBatchRunStopReceived
		}
		// Fail has upper hand over stopped
		if failFound {
			nodeStatus = NodeBatchFail
		}
		return nodeStatus, len(batchesInProgress), int(foundBatchesTotal), nil
	}

	// Some batches are still not complete, consider it in progress until all batches are complete (via success/fail/stop)
	return NodeBatchStart, len(batchesInProgress), int(foundBatchesTotal), nil
}
