package wfmodel

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

type NodeStatusType int8

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
		sb.WriteString(fmt.Sprintf(`"%s":"%s"`, nodeName, nodeStatus.ToString()))
	}
	sb.WriteString("}")
	return sb.String()
}

type RunBatchStatusMap map[int16]NodeBatchStatusType

func (m RunBatchStatusMap) ToString() string {
	sb := strings.Builder{}
	sb.WriteString("{")
	for runId, nodeStatus := range m {
		sb.WriteString(fmt.Sprintf("%d:%s ", runId, nodeStatus.ToString()))
	}
	sb.WriteString("}")
	return sb.String()
}

type NodeRunBatchStatusMap map[string]RunBatchStatusMap

func (m NodeRunBatchStatusMap) ToString() string {
	sb := strings.Builder{}
	sb.WriteString("{")
	for nodeName, runBatchStatusMap := range m {
		sb.WriteString(fmt.Sprintf("%s:%s ", nodeName, runBatchStatusMap.ToString()))
	}
	sb.WriteString("}")
	return sb.String()
}

// Object model with tags that allow to create cql CREATE TABLE queries and to print object
type NodeHistoryEvent struct {
	Ts         time.Time           `header:"ts" format:"%-33v" column:"ts" type:"timestamp" json:"ts"`
	RunId      int16               `header:"run_id" format:"%6d" column:"run_id" type:"int" key:"true" json:"run_id"`
	ScriptNode string              `header:"script_node" format:"%20v" column:"script_node" type:"text" key:"true" json:"script_node"`
	Status     NodeBatchStatusType `header:"sts" format:"%3v" column:"status" type:"tinyint" key:"true" json:"status"`
	Comment    string              `header:"comment" format:"%v" column:"comment" type:"text" json:"comment"`
}

func NodeHistoryEventAllFields() []string {
	return []string{"ts", "run_id", "script_node", "status", "comment"}
}
func NewNodeHistoryEventFromMap(r map[string]interface{}, fields []string) (*NodeHistoryEvent, error) {
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

// ToSpacedString - prints formatted field values, uses reflection, shoud not be used in prod
func (n NodeHistoryEvent) ToSpacedString() string {
	t := reflect.TypeOf(n)
	formats := GetObjectModelFieldFormats(t)
	values := make([]string, t.NumField())

	v := reflect.ValueOf(&n).Elem()
	for i := 0; i < v.NumField(); i++ {
		fv := v.Field(i)
		values[i] = fmt.Sprintf(formats[i], fv)
	}
	return strings.Join(values, PrintTableDelimiter)
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
