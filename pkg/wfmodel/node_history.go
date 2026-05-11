package wfmodel

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

const TableNameNodeHistory = "wf_node_history"

/*
const (
	NodeNone           NodeBatchStatusType = 0
	NodeStart           NodeBatchStatusType = 1
	NodeSuccess         NodeBatchStatusType = 2
	NodeFail            NodeBatchStatusType = 3
	NodeRunStopReceived NodeBatchStatusType = 104
)

func (status NodeBatchStatusType) ToString() string {
	switch status {
	case NodeNone:
		return "none"
	case NodeStart:
		return "start"
	case NodeSuccess:
		return "success"
	case NodeFail:
		return "fail"
	case NodeRunStopReceived:
		return "stopreceived"
	default:
		return "unknown"
	}
}
*/

type NodeStatusMap map[string]NodeBatchStatusType

func (m NodeStatusMap) ToString() string {
	sb := strings.Builder{}
	sb.WriteString("{")
	for nodeName, nodeStatus := range m {
		if sb.Len() > 1 {
			sb.WriteString(",")
		}
		fmt.Fprintf(&sb, `"%s":"%s"`, nodeName, NodeBatchStatusToString(nodeStatus))
	}
	sb.WriteString("}")
	return sb.String()
}

// Object model with tags that allow to create cql CREATE TABLE queries and to print object
type NodeHistoryEvent struct {
	Ts                time.Time           `header:"ts" format:"%-33v" column:"ts" type:"timestamp" json:"ts"`
	RunId             int16               `header:"run_id" format:"%6d" column:"run_id" type:"int" key:"true" json:"run_id"` // Partitioning key - we use it in WHERE
	ScriptNode        string              `header:"script_node" format:"%20v" column:"script_node" type:"text" key:"true" json:"script_node"`
	WrittenByBatchIdx int16               `header:"written_by_batch_idx" format:"%5v" column:"written_by_batch_idx" type:"int" key:"true" json:"written_by_batch_idx"`
	Status            NodeBatchStatusType `header:"sts" format:"%3v" column:"status" type:"tinyint" key:"true" json:"status"`
	Comment           string              `header:"comment" format:"%v" column:"comment" type:"text" json:"comment"`
}

func NodeHistoryEventAllFields() []string {
	return []string{"ts", "run_id", "script_node", "written_by_batch_idx", "status", "comment"}
}
func NewNodeHistoryEventFromMap(r map[string]any, fields []string) (*NodeHistoryEvent, error) {
	res := &NodeHistoryEvent{}
	for _, fieldName := range fields {
		var err error
		switch fieldName {
		case "ts":
			res.Ts, err = ReadTimeFromRow(fieldName, r)
		case "run_id":
			res.RunId, err = ReadInt16FromRow(fieldName, r)
		case "script_node":
			res.ScriptNode, err = ReadStringFromRow(fieldName, r)
		case "written_by_batch_idx":
			res.WrittenByBatchIdx, err = ReadInt16FromRow(fieldName, r)
		case "status":
			res.Status, err = ReadNodeBatchStatusFromRow(fieldName, r)
		case "comment":
			res.Comment, err = ReadStringFromRow(fieldName, r)
		default:
			return nil, fmt.Errorf("unknown %s field %s", fieldName, TableNameNodeHistory)
		}
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

func appendNodeEventsFromMap(nodeEventMaps map[int16]map[string]*NodeHistoryEvent, nodeEvents []*NodeHistoryEvent) []*NodeHistoryEvent {
	for _, nodeMap := range nodeEventMaps {
		for _, rec := range nodeMap {
			nodeEvents = append(nodeEvents, rec)
		}
	}
	return nodeEvents
}

func earliestEventToNodeMap(rec *NodeHistoryEvent, nodeEventMaps map[int16]map[string]*NodeHistoryEvent) {
	if _, ok := nodeEventMaps[rec.RunId]; !ok {
		nodeEventMaps[rec.RunId] = map[string]*NodeHistoryEvent{}
	}
	runNodeEvent, ok := nodeEventMaps[rec.RunId][rec.ScriptNode]
	if !ok || ok && runNodeEvent.Ts.After(rec.Ts) {
		nodeEventMaps[rec.RunId][rec.ScriptNode] = rec
	}
}

func lastEventToNodeMap(rec *NodeHistoryEvent, nodeEventMaps map[int16]map[string]*NodeHistoryEvent) {
	if _, ok := nodeEventMaps[rec.RunId]; !ok {
		nodeEventMaps[rec.RunId] = map[string]*NodeHistoryEvent{}
	}
	runNodeEvent, ok := nodeEventMaps[rec.RunId][rec.ScriptNode]
	if !ok || ok && runNodeEvent.Ts.Before(rec.Ts) {
		nodeEventMaps[rec.RunId][rec.ScriptNode] = rec
	}
}

// Converts returned rows to a slice of events, removes duplicates and sorts them by ts/run_id/written_by_batch_idx
func NodeHistoryRowsToEvents(rows []map[string]any) ([]*NodeHistoryEvent, error) {
	// Here we pay for our decision to save multiple "node start" events, one for each batch, to avoid contention errors.
	// The caller is only interested in the first "node start" event, ignore the other, so we can avoid managing thousands of useless records
	nodeStartMap := map[int16]map[string]*NodeHistoryEvent{}
	nodeSuccessMap := map[int16]map[string]*NodeHistoryEvent{}
	nodeFailMap := map[int16]map[string]*NodeHistoryEvent{}
	nodeStopMap := map[int16]map[string]*NodeHistoryEvent{}

	fields := NodeHistoryEventAllFields()
	for _, r := range rows {
		rec, err := NewNodeHistoryEventFromMap(r, fields)
		if err != nil {
			return nil, fmt.Errorf("cannot deserialize node history row %v: %s", r, err.Error())
		}
		switch rec.Status {
		case NodeBatchStart:
			// Save the earliest "node start" for run/node in the map, but do not add it to nodeEvents
			earliestEventToNodeMap(rec, nodeStartMap)
		case NodeBatchSuccess:
			// Save the latest "node success" for run/node in the map, but do not add it to nodeEvents
			lastEventToNodeMap(rec, nodeSuccessMap)
		case NodeBatchFail:
			// Save the latest "node fail" for run/node in the map, but do not add it to nodeEvents
			lastEventToNodeMap(rec, nodeFailMap)
		case NodeBatchRunStopReceived:
			// Save the latest "node run stopped" for run/node in the map, but do not add it to nodeEvents
			lastEventToNodeMap(rec, nodeStopMap)
		default:
			// NodeBatchNone, Do nothing
		}
	}

	// Collect events from maps into nodeEvents
	nodeEvents := make([]*NodeHistoryEvent, 0)
	nodeEvents = appendNodeEventsFromMap(nodeStartMap, nodeEvents)
	nodeEvents = appendNodeEventsFromMap(nodeSuccessMap, nodeEvents)
	nodeEvents = appendNodeEventsFromMap(nodeFailMap, nodeEvents)
	nodeEvents = appendNodeEventsFromMap(nodeStopMap, nodeEvents)

	slices.SortFunc(nodeEvents, func(l, r *NodeHistoryEvent) int {
		switch {
		case l.Ts.Before(r.Ts):
			return -1
		case l.Ts.After(r.Ts):
			return 1
		default:
			switch {
			case l.RunId < r.RunId:
				return -1
			case l.RunId > r.RunId:
				return 1
			default:
				switch {
				case l.WrittenByBatchIdx < r.WrittenByBatchIdx:
					return -1
				case l.WrittenByBatchIdx > r.WrittenByBatchIdx:
					return 1
				default:
					return -1
				}
			}
		}
	})

	return nodeEvents, nil
}

// From multiple node events, decide the status by priority: stop > fail > success > start
func FigureOutRunStatusAndAffectedNodesStatusesFromNodeEvents(sortedNodeEvents []*NodeHistoryEvent, runId int16, affectedNodes []string) (NodeBatchStatusType, NodeStatusMap) {
	// For each affected node of this run, figure out its status
	nodeStatusMap := NodeStatusMap{}
	for _, affectedNodeName := range affectedNodes {
		nodeStatusMap[affectedNodeName] = NodeBatchNone
	}

	for _, e := range sortedNodeEvents {
		if e.RunId != runId {
			continue
		}
		lastNodeStatusSoFar, ok := nodeStatusMap[e.ScriptNode]
		if !ok {
			nodeStatusMap[e.ScriptNode] = e.Status
		} else {
			if lastNodeStatusSoFar != NodeBatchRunStopReceived {
				nodeStatusMap[e.ScriptNode] = e.Status
			}
			if e.Status == NodeBatchRunStopReceived && lastNodeStatusSoFar != NodeBatchRunStopReceived {
				nodeStatusMap[e.ScriptNode] = NodeBatchRunStopReceived
			} else if e.Status == NodeBatchFail && lastNodeStatusSoFar != NodeBatchFail && lastNodeStatusSoFar != NodeBatchRunStopReceived {
				nodeStatusMap[e.ScriptNode] = NodeBatchFail
			} else if e.Status == NodeBatchSuccess && lastNodeStatusSoFar != NodeBatchSuccess && lastNodeStatusSoFar != NodeBatchFail && lastNodeStatusSoFar != NodeBatchRunStopReceived {
				nodeStatusMap[e.ScriptNode] = NodeBatchSuccess
			} else if e.Status == NodeBatchStart && lastNodeStatusSoFar != NodeBatchStart && lastNodeStatusSoFar != NodeBatchSuccess && lastNodeStatusSoFar != NodeBatchFail && lastNodeStatusSoFar != NodeBatchRunStopReceived {
				nodeStatusMap[e.ScriptNode] = NodeBatchStart
			}
		}
	}

	highestStatus := NodeBatchNone
	lowestStatus := NodeBatchRunStopReceived
	for _, status := range nodeStatusMap {
		if status > highestStatus {
			highestStatus = status
		}
		if status < lowestStatus {
			lowestStatus = status
		}
	}

	// If all affected nodes are success/fail/stopped, return the highest of the success/fail/stopped statuses
	if lowestStatus > NodeBatchStart {
		return highestStatus, nodeStatusMap
	}

	// Some of the affected nodes are none/started, returtn the lowest of none/started
	return lowestStatus, nodeStatusMap
}
