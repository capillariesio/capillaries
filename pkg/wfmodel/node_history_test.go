package wfmodel

import (
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func (e *NodeHistoryEvent) ToMap() map[string]any {
	m := map[string]any{}
	for _, fieldName := range NodeHistoryEventAllFields() {
		switch fieldName {
		case "ts":
			m[fieldName] = e.Ts
		case "run_id":
			m[fieldName] = e.RunId
		case "script_node":
			m[fieldName] = e.ScriptNode
		case "written_by_batch_idx":
			m[fieldName] = e.WrittenByBatchIdx
		case "status":
			m[fieldName] = int8(e.Status) // Pretend we return it from Cassandra which uses int8 there
		case "comment":
			m[fieldName] = e.Comment
		default:
		}
	}
	return m
}

func TestNodeHistoryRowsToEvents(t *testing.T) {
	rows := []map[string]any{
		// Run 1
		(&NodeHistoryEvent{
			Ts:                time.Date(2001, 1, 1, 1, 1, 1, 0, time.UTC),
			RunId:             int16(1),
			ScriptNode:        "node1",
			WrittenByBatchIdx: 0,
			Status:            NodeStart,
		}).ToMap(),
		(&NodeHistoryEvent{
			Ts:                time.Date(2001, 1, 1, 1, 1, 2, 0, time.UTC),
			RunId:             int16(1),
			ScriptNode:        "node1",
			WrittenByBatchIdx: 1,
			Status:            NodeStart,
		}).ToMap(),
		(&NodeHistoryEvent{
			Ts:                time.Date(2001, 1, 1, 1, 1, 3, 0, time.UTC),
			RunId:             int16(1),
			ScriptNode:        "node1",
			WrittenByBatchIdx: 1,
			Status:            NodeSuccess,
		}).ToMap(),
		(&NodeHistoryEvent{
			Ts:                time.Date(2001, 1, 1, 1, 1, 4, 0, time.UTC),
			RunId:             int16(1),
			ScriptNode:        "node1",
			WrittenByBatchIdx: 100,
			Status:            NodeRunStopReceived,
		}).ToMap(),
		(&NodeHistoryEvent{
			Ts:                time.Date(2001, 1, 1, 1, 1, 5, 0, time.UTC),
			RunId:             int16(1),
			ScriptNode:        "node1",
			WrittenByBatchIdx: 101,
			Status:            NodeRunStopReceived,
		}).ToMap(),

		// Run 2

		(&NodeHistoryEvent{
			Ts:                time.Date(2002, 1, 1, 1, 1, 1, 0, time.UTC),
			RunId:             int16(2),
			ScriptNode:        "node1",
			WrittenByBatchIdx: 0,
			Status:            NodeStart,
		}).ToMap(),
		(&NodeHistoryEvent{
			Ts:                time.Date(2002, 1, 1, 1, 1, 2, 0, time.UTC),
			RunId:             int16(2),
			ScriptNode:        "node1",
			WrittenByBatchIdx: 1,
			Status:            NodeStart,
		}).ToMap(),
		(&NodeHistoryEvent{
			Ts:                time.Date(2002, 1, 1, 1, 1, 3, 0, time.UTC),
			RunId:             int16(2),
			ScriptNode:        "node1",
			WrittenByBatchIdx: 1,
			Status:            NodeFail,
		}).ToMap(),
		(&NodeHistoryEvent{
			Ts:                time.Date(2002, 1, 1, 1, 1, 4, 0, time.UTC),
			RunId:             int16(2),
			ScriptNode:        "node1",
			WrittenByBatchIdx: 100,
			Status:            NodeRunStopReceived,
		}).ToMap(),
		(&NodeHistoryEvent{
			Ts:                time.Date(2002, 1, 1, 1, 1, 5, 0, time.UTC),
			RunId:             int16(2),
			ScriptNode:        "node1",
			WrittenByBatchIdx: 101,
			Status:            NodeRunStopReceived,
		}).ToMap(),
	}

	events, err := NodeHistoryRowsToEvents(rows)
	assert.Nil(t, err)
	assert.Equal(t, 6, len(events))
	assert.Equal(t, "{2001-01-01 01:01:01 +0000 UTC 1 node1 0 1 }", fmt.Sprintf("%v", *events[0]))
	assert.Equal(t, "{2001-01-01 01:01:03 +0000 UTC 1 node1 1 2 }", fmt.Sprintf("%v", *events[1]))
	assert.Equal(t, "{2001-01-01 01:01:05 +0000 UTC 1 node1 101 104 }", fmt.Sprintf("%v", *events[2]))
	assert.Equal(t, "{2002-01-01 01:01:01 +0000 UTC 2 node1 0 1 }", fmt.Sprintf("%v", *events[3]))
	assert.Equal(t, "{2002-01-01 01:01:03 +0000 UTC 2 node1 1 3 }", fmt.Sprintf("%v", *events[4]))
	assert.Equal(t, "{2002-01-01 01:01:05 +0000 UTC 2 node1 101 104 }", fmt.Sprintf("%v", *events[5]))
}

func TestFigureOutRunStatusAndAffectedNodesStatusesFromNodeEvents(t *testing.T) {
	events := []*NodeHistoryEvent{
		// Node 1
		(&NodeHistoryEvent{
			Ts:         time.Date(2001, 1, 1, 1, 1, 1, 0, time.UTC),
			RunId:      int16(1),
			ScriptNode: "node1",
			Status:     NodeStart,
		}),
		(&NodeHistoryEvent{
			Ts:         time.Date(2001, 1, 1, 1, 1, 2, 0, time.UTC),
			RunId:      int16(1),
			ScriptNode: "node1",
			Status:     NodeSuccess,
		}),
		(&NodeHistoryEvent{
			Ts:         time.Date(2001, 1, 1, 1, 1, 3, 0, time.UTC),
			RunId:      int16(1),
			ScriptNode: "node1",
			Status:     NodeRunStopReceived,
		}),

		// Node 2

		(&NodeHistoryEvent{
			Ts:         time.Date(2001, 1, 1, 1, 1, 4, 0, time.UTC),
			RunId:      int16(1),
			ScriptNode: "node2",
			Status:     NodeStart,
		}),
		(&NodeHistoryEvent{
			Ts:                time.Date(2001, 1, 1, 1, 1, 5, 0, time.UTC),
			RunId:             int16(1),
			ScriptNode:        "node2",
			WrittenByBatchIdx: 1,
			Status:            NodeFail,
		}),
		(&NodeHistoryEvent{
			Ts:         time.Date(2001, 1, 1, 1, 1, 6, 0, time.UTC),
			RunId:      int16(1),
			ScriptNode: "node2",
			Status:     NodeRunStopReceived,
		}),
	}

	// Stop received twice: run node status is "stopped"
	nodeBatchStatusType, nodeStatusMap := FigureOutRunStatusAndAffectedNodesStatusesFromNodeEvents(events, int16(1), []string{"node1", "node2"})
	assert.Equal(t, NodeBatchRunStopReceived, nodeBatchStatusType)
	assert.Equal(t, NodeBatchRunStopReceived, nodeStatusMap["node1"])
	assert.Equal(t, NodeBatchRunStopReceived, nodeStatusMap["node2"])

	// Remove node2 stop: run node status is "stopped"
	events = slices.Delete(events, len(events)-1, len(events))
	nodeBatchStatusType, nodeStatusMap = FigureOutRunStatusAndAffectedNodesStatusesFromNodeEvents(events, int16(1), []string{"node1", "node2"})
	assert.Equal(t, NodeBatchRunStopReceived, nodeBatchStatusType)
	assert.Equal(t, NodeBatchRunStopReceived, nodeStatusMap["node1"])
	assert.Equal(t, NodeBatchFail, nodeStatusMap["node2"])

	// Remove node1 stop: run node status is "fail"
	events = slices.Delete(events, 2, 3)
	nodeBatchStatusType, nodeStatusMap = FigureOutRunStatusAndAffectedNodesStatusesFromNodeEvents(events, int16(1), []string{"node1", "node2"})
	assert.Equal(t, NodeBatchFail, nodeBatchStatusType)
	assert.Equal(t, NodeBatchSuccess, nodeStatusMap["node1"])
	assert.Equal(t, NodeBatchFail, nodeStatusMap["node2"])

	// Remove node2 fail: run node status is "start"
	events = slices.Delete(events, 3, 4)
	nodeBatchStatusType, nodeStatusMap = FigureOutRunStatusAndAffectedNodesStatusesFromNodeEvents(events, int16(1), []string{"node1", "node2"})
	assert.Equal(t, NodeBatchStart, nodeBatchStatusType)
	assert.Equal(t, NodeBatchSuccess, nodeStatusMap["node1"])
	assert.Equal(t, NodeBatchStart, nodeStatusMap["node2"])

	// Remove node2 start: no more node2 events so run node status is "none"
	events = slices.Delete(events, 2, 3)
	nodeBatchStatusType, nodeStatusMap = FigureOutRunStatusAndAffectedNodesStatusesFromNodeEvents(events, int16(1), []string{"node1", "node2"})
	assert.Equal(t, NodeBatchNone, nodeBatchStatusType)
	assert.Equal(t, NodeBatchSuccess, nodeStatusMap["node1"])
	assert.Equal(t, NodeBatchNone, nodeStatusMap["node2"])
}
