package wfmodel

import (
	"fmt"
	"strings"
	"time"
)

type RunStatusType int8

const (
	RunNone     RunStatusType = 0
	RunStart    RunStatusType = 1
	RunComplete RunStatusType = 2
	RunStop     RunStatusType = 3
)

const TableNameRunHistory = "wf_run_history"

func (status RunStatusType) ToString() string {
	switch status {
	case RunNone:
		return "none"
	case RunStart:
		return "start"
	case RunComplete:
		return "complete"
	case RunStop:
		return "stop"
	default:
		return "unknown"
	}
}

type RunStatusMap map[int16]RunStatusType
type RunStartTsMap map[int16]time.Time

func (m RunStartTsMap) ToString() string {
	sb := strings.Builder{}
	for runId, ts := range m {
		sb.WriteString(fmt.Sprintf("%d:%s,", runId, ts.Format(LogTsFormatQuoted)))
	}
	return sb.String()
}

func (m RunStatusMap) ToString() string {
	sb := strings.Builder{}
	for runId, runStatus := range m {
		sb.WriteString(fmt.Sprintf("%d:%s,", runId, runStatus.ToString()))
	}
	return sb.String()
}

// Object model with tags that allow to create cql CREATE TABLE queries and to print object
type RunHistoryEvent struct {
	Ts      time.Time     `header:"ts" format:"%-33v" column:"ts" type:"timestamp" json:"ts"`
	RunId   int16         `header:"run_id" format:"%6d" column:"run_id" type:"int" key:"true" json:"run_id"`
	Status  RunStatusType `header:"sts" format:"%3v" column:"status" type:"tinyint" key:"true" json:"status"`
	Comment string        `header:"comment" format:"%v" column:"comment" type:"text" json:"comment"`
}

func RunHistoryEventAllFields() []string {
	return []string{"ts", "run_id", "status", "comment"}
}

func NewRunHistoryEventFromMap(r map[string]any, fields []string) (*RunHistoryEvent, error) {
	res := &RunHistoryEvent{}
	for _, fieldName := range fields {
		var err error
		switch fieldName {
		case "ts":
			res.Ts, err = ReadTimeFromRow(fieldName, r)
		case "run_id":
			res.RunId, err = ReadInt16FromRow(fieldName, r)
		case "status":
			res.Status, err = ReadRunStatusFromRow(fieldName, r)
		case "comment":
			res.Comment, err = ReadStringFromRow(fieldName, r)
		default:
			return nil, fmt.Errorf("unknown %s field %s", fieldName, TableNameRunHistory)
		}
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

// ToSpacedString - prints formatted field values, uses reflection, shoud not be used in prod
// func (n RunHistoryEvent) ToSpacedString() string {
// 	t := reflect.TypeOf(n)
// 	formats := GetObjectModelFieldFormats(t)
// 	values := make([]string, t.NumField())

// 	v := reflect.ValueOf(&n).Elem()
// 	for i := 0; i < v.NumField(); i++ {
// 		fv := v.Field(i)
// 		values[i] = fmt.Sprintf(formats[i], fv)
// 	}
// 	return strings.Join(values, PrintTableDelimiter)
// }

type RunLifespan struct {
	RunId            int16         `json:"run_id"`
	StartTs          time.Time     `json:"start_ts"`
	StartComment     string        `json:"start_comment"`
	FinalStatus      RunStatusType `json:"final_status"`
	CompletedTs      time.Time     `json:"completed_ts"`
	CompletedComment string        `json:"completed_comment"`
	StoppedTs        time.Time     `json:"stopped_ts"`
	StoppedComment   string        `json:"stopped_comment"`
}

func (ls RunLifespan) ToString() string {
	return fmt.Sprintf("{run_id: %d, start_ts:%s, final_status:%s, completed_ts:%s, stopped_ts:%s}",
		ls.RunId,
		ls.StartTs.Format(LogTsFormatQuoted),
		ls.FinalStatus.ToString(),
		ls.CompletedTs.Format(LogTsFormatQuoted),
		ls.StoppedTs.Format(LogTsFormatQuoted))
}

type RunLifespanMap map[int16]*RunLifespan

func (m RunLifespanMap) ToString() string {
	items := make([]string, len(m))
	itemIdx := 0
	for runId, ls := range m {
		items[itemIdx] = fmt.Sprintf("%d:%s", runId, ls.ToString())
		itemIdx++
	}
	return fmt.Sprintf("{%s}", strings.Join(items, ", "))
}

// func InheritNodeBatchStatusToRunStatus(nodeBatchStatus NodeBatchStatusType) (RunStatusType, error) {
// 	switch nodeBatchStatus {
// 	case NodeBatchFail:
// 		return RunFail, nil
// 	case NodeBatchSuccess:
// 		return RunSuccess, nil
// 	default:
// 		return RunNone, fmt.Errorf("cannot inherit run from node batch status %d", nodeBatchStatus)
// 	}
// }
