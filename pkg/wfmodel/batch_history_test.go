package wfmodel

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func (e *BatchHistoryEvent) ToMap() map[string]any {
	m := map[string]any{}
	for _, fieldName := range BatchHistoryEventAllFields() {
		switch fieldName {
		case "ts":
			m[fieldName] = e.Ts
		case "run_id":
			m[fieldName] = e.RunId
		case "script_node":
			m[fieldName] = e.ScriptNode
		case "batch_idx":
			m[fieldName] = e.BatchIdx
		case "batches_total":
			m[fieldName] = e.BatchesTotal
		case "status":
			m[fieldName] = int8(e.Status) // Pretend this is returned by Cassandra
		case "first_token":
			m[fieldName] = e.FirstToken
		case "last_token":
			m[fieldName] = e.LastToken
		case "instance":
			m[fieldName] = e.Instance
		case "thread":
			m[fieldName] = e.Thread
		case "comment":
			m[fieldName] = e.Comment
		default:
		}
	}
	return m
}

func TestBatchHistoryRowsToEventsGood(t *testing.T) {
	rows := []map[string]any{
		(&BatchHistoryEvent{
			Ts:           time.Date(2001, 1, 1, 1, 1, 2, 0, time.UTC),
			RunId:        int16(2),
			ScriptNode:   "node1",
			BatchIdx:     int16(0),
			BatchesTotal: 1,
			Status:       NodeBatchStart,
			FirstToken:   int64(0),
			LastToken:    int64(1000000),
			Instance:     "inst1",
			Thread:       int64(1234),
		}).ToMap(),
		(&BatchHistoryEvent{
			Ts:           time.Date(2001, 1, 1, 1, 1, 1, 0, time.UTC),
			RunId:        int16(1),
			ScriptNode:   "node1",
			BatchIdx:     int16(0),
			BatchesTotal: 1,
			Status:       NodeBatchStart,
			FirstToken:   int64(0),
			LastToken:    int64(1000000),
			Instance:     "inst1",
			Thread:       int64(1234),
		}).ToMap(),
	}

	events, err := BatchHistoryRowsToEvents(rows)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(events))
	assert.Equal(t, "{2001-01-01 01:01:01 +0000 UTC 1 node1 0 1 1 0 1000000 inst1 1234 }", fmt.Sprintf("%v", *events[0]))
	assert.Equal(t, "{2001-01-01 01:01:02 +0000 UTC 2 node1 0 1 1 0 1000000 inst1 1234 }", fmt.Sprintf("%v", *events[1]))
}

func TestBatchHistoryRowsToEventsBad(t *testing.T) {
	rows := []map[string]any{
		(&BatchHistoryEvent{
			Ts:           time.Date(2001, 1, 1, 1, 1, 2, 0, time.UTC),
			RunId:        int16(2),
			ScriptNode:   "node1",
			BatchIdx:     int16(0),
			BatchesTotal: 1,
			Status:       NodeBatchStart,
			FirstToken:   int64(0),
			LastToken:    int64(1000000),
			Instance:     "inst1",
			Thread:       int64(1234),
		}).ToMap(),
	}

	rows[0]["ts"] = "a"

	_, err := BatchHistoryRowsToEvents(rows)
	assert.Contains(t, err.Error(), "cannot read time ts")
}

func TestNodeBatchStatusToString(t *testing.T) {
	assert.Equal(t, "none", NodeBatchStatusToString(NodeBatchNone))
	assert.Equal(t, "start", NodeBatchStatusToString(NodeBatchStart))
	assert.Equal(t, "success", NodeBatchStatusToString(NodeBatchSuccess))
	assert.Equal(t, "fail", NodeBatchStatusToString(NodeBatchFail))
	assert.Equal(t, "stopreceived", NodeBatchStatusToString(NodeBatchRunStopReceived))
	assert.Equal(t, "unknown", NodeBatchStatusToString(100))
}

func TestBatchHistoryRowsToNodeStatus(t *testing.T) {
	rows := []map[string]any{
		(&BatchHistoryEvent{
			Ts:           time.Date(2001, 1, 1, 1, 1, 1, 0, time.UTC),
			BatchIdx:     int16(0),
			BatchesTotal: 2,
			Status:       NodeBatchStart,
		}).ToMap(),
		(&BatchHistoryEvent{
			Ts:           time.Date(2001, 1, 1, 1, 1, 2, 0, time.UTC),
			BatchIdx:     int16(0),
			BatchesTotal: 2,
			Status:       NodeBatchFail,
		}).ToMap(),
		(&BatchHistoryEvent{
			Ts:           time.Date(2001, 1, 1, 1, 1, 3, 0, time.UTC),
			BatchIdx:     int16(1),
			BatchesTotal: 2,
			Status:       NodeBatchStart,
		}).ToMap(),
		(&BatchHistoryEvent{
			Ts:           time.Date(2001, 1, 1, 1, 1, 4, 0, time.UTC),
			BatchIdx:     int16(1),
			BatchesTotal: 2,
			Status:       NodeBatchSuccess,
		}).ToMap(),
	}

	// First batch fail: result fail
	nodeStatus, batchesInProgress, batchesTotal, err := BatchHistoryRowsToNodeStatus(rows)
	assert.Nil(t, err)
	assert.Equal(t, NodeBatchFail, nodeStatus)
	assert.Equal(t, 0, batchesInProgress)
	assert.Equal(t, 2, batchesTotal)

	// First batch stopped: result stopped
	rows[1]["status"] = int8(NodeBatchRunStopReceived)
	nodeStatus, batchesInProgress, batchesTotal, err = BatchHistoryRowsToNodeStatus(rows)
	assert.Nil(t, err)
	assert.Equal(t, NodeBatchRunStopReceived, nodeStatus)
	assert.Equal(t, 0, batchesInProgress)
	assert.Equal(t, 2, batchesTotal)

	// Second batch fail: result fail
	rows[3]["status"] = int8(NodeBatchFail)
	nodeStatus, batchesInProgress, batchesTotal, err = BatchHistoryRowsToNodeStatus(rows)
	assert.Nil(t, err)
	assert.Equal(t, NodeBatchFail, nodeStatus)
	assert.Equal(t, 0, batchesInProgress)
	assert.Equal(t, 2, batchesTotal)

	// Both just started: result started
	rows[1]["status"] = int8(NodeBatchStart)
	rows[3]["status"] = int8(NodeBatchStart)
	nodeStatus, batchesInProgress, batchesTotal, err = BatchHistoryRowsToNodeStatus(rows)
	assert.Nil(t, err)
	assert.Equal(t, NodeBatchStart, nodeStatus)
	assert.Equal(t, 2, batchesInProgress)
	assert.Equal(t, 2, batchesTotal)

	// First start, second stop: result started
	rows[3]["status"] = int8(NodeBatchRunStopReceived)
	nodeStatus, batchesInProgress, batchesTotal, err = BatchHistoryRowsToNodeStatus(rows)
	assert.Nil(t, err)
	assert.Equal(t, NodeBatchStart, nodeStatus)
	assert.Equal(t, 1, batchesInProgress)
	assert.Equal(t, 2, batchesTotal)

	// Bad input

	rows[3]["status"] = "a"
	_, _, _, err = BatchHistoryRowsToNodeStatus(rows)
	assert.Contains(t, err.Error(), "cannot read node/batch status")

	rows[3]["status"] = int8(NodeBatchRunStopReceived)
	rows[0]["batches_total"] = 100
	_, _, _, err = BatchHistoryRowsToNodeStatus(rows)
	assert.Contains(t, err.Error(), "conflicting batches total value, was 100, now 2")

	rows[0]["batches_total"] = 2
	rows[3]["batch_idx"] = 2
	_, _, _, err = BatchHistoryRowsToNodeStatus(rows)
	assert.Contains(t, err.Error(), "invalid batch idx/total(2/2)")
}
