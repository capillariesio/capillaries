package wfmodel

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

const (
	NodehNone           NodeBatchStatusType = 0
	NodeStart           NodeBatchStatusType = 1
	NodeSuccess         NodeBatchStatusType = 2
	NodeFail            NodeBatchStatusType = 3
	NodeRunStopReceived NodeBatchStatusType = 104
)

const TableNameNodeHistory = "wf_node_history"

func (status NodeBatchStatusType) ToString() string {
	switch status {
	case NodehNone:
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

type NodeStatusMap map[string]NodeBatchStatusType

func (m NodeStatusMap) ToString() string {
	sb := strings.Builder{}
	sb.WriteString("{")
	for nodeName, nodeStatus := range m {
		if sb.Len() > 1 {
			sb.WriteString(",")
		}
		fmt.Fprintf(&sb, `"%s":"%s"`, nodeName, nodeStatus.ToString())
	}
	sb.WriteString("}")
	return sb.String()
}

type RunBatchStatusMap map[int16]NodeBatchStatusType

func (m RunBatchStatusMap) ToString() string {
	sb := strings.Builder{}
	sb.WriteString("{")
	for runId, nodeStatus := range m {
		fmt.Fprintf(&sb, "%d:%s ", runId, nodeStatus.ToString())
	}
	sb.WriteString("}")
	return sb.String()
}

type NodeRunBatchStatusMap map[string]RunBatchStatusMap

func (m NodeRunBatchStatusMap) ToString() string {
	sb := strings.Builder{}
	sb.WriteString("{")
	for nodeName, runBatchStatusMap := range m {
		fmt.Fprintf(&sb, "%s:%s ", nodeName, runBatchStatusMap.ToString())
	}
	sb.WriteString("}")
	return sb.String()
}

// Object model with tags that allow to create cql CREATE TABLE queries and to print object
type NodeHistoryEvent struct {
	Ts                time.Time           `header:"ts" format:"%-33v" column:"ts" type:"timestamp" json:"ts"`
	RunId             int16               `header:"run_id" format:"%6d" column:"run_id" type:"int" key:"true" json:"run_id"`
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

// Converts returned rows to a slice of events and sorts them by ts/run_id/written_by_batch_idx
func NodeHistoryRowsToNodeHistoryEvents(rows []map[string]any, fields []string) ([]*NodeHistoryEvent, error) {
	nodeEvents := make([]*NodeHistoryEvent, len(rows))

	for idx, r := range rows {
		rec, err := NewNodeHistoryEventFromMap(r, fields)
		if err != nil {
			return nodeEvents, fmt.Errorf("cannot deserialize node history row: %s", err.Error())
		}
		nodeEvents[idx] = rec
	}

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

type NodeLifespan struct {
	StartTs      time.Time
	LastStatus   NodeBatchStatusType
	LastStatusTs time.Time
}

func (ls NodeLifespan) ToString() string {
	return fmt.Sprintf("{start_ts:%s, last_status:%s, last_status_ts:%s}",
		ls.StartTs.Format(LogTsFormatQuoted),
		ls.LastStatus.ToString(),
		ls.LastStatusTs.Format(LogTsFormatQuoted))
}

type NodeLifespanMap map[string]*NodeLifespan

func (m NodeLifespanMap) ToString() string {
	items := make([]string, len(m))
	nodeIdx := 0
	for nodeName, ls := range m {
		items[nodeIdx] = fmt.Sprintf("%s:%s", nodeName, ls.ToString())
		nodeIdx++
	}
	return fmt.Sprintf("{%s}", strings.Join(items, ", "))
}

type RunNodeLifespanMap map[int16]NodeLifespanMap

func (m RunNodeLifespanMap) ToString() string {
	items := make([]string, len(m))
	runIdx := 0
	for runId, ls := range m {
		items[runIdx] = fmt.Sprintf("%d:%s", runId, ls.ToString())
		runIdx++
	}
	return fmt.Sprintf("{%s}", strings.Join(items, ", "))
}
