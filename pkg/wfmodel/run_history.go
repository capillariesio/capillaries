package wfmodel

import (
	"fmt"
	"slices"
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

// Object model with tags that allow to create cql CREATE TABLE queries and to print object
type RunHistoryEvent struct {
	Ts      time.Time     `header:"ts" format:"%-33v" column:"ts" type:"timestamp" json:"ts"`
	RunId   int16         `header:"run_id" format:"%6d" column:"run_id" type:"int" key:"true" json:"run_id"` // Partitioning key, used in IN()
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
	return fmt.Sprintf("{run_id: %d, start_ts:%s, final_status:%s, completed_ts:%s, stopped_ts:%s, completed_comment:%s, stopped_comment:%s}",
		ls.RunId,
		ls.StartTs.Format(LogTsFormatQuoted),
		ls.FinalStatus.ToString(),
		ls.CompletedTs.Format(LogTsFormatQuoted),
		ls.StoppedTs.Format(LogTsFormatQuoted),
		ls.CompletedComment,
		ls.StoppedComment,
	)
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

func RunHistoryEventsToRunStatusMap(sortedRunHistoryEvents []*RunHistoryEvent) map[int16]RunStatusType {
	runStatusMap := map[int16]RunStatusType{}
	for _, e := range sortedRunHistoryEvents {
		curRunStatus, ok := runStatusMap[e.RunId]
		if !ok {
			runStatusMap[e.RunId] = e.Status
		} else {
			if e.Status == RunStop {
				runStatusMap[e.RunId] = RunStop
			} else if e.Status == RunComplete && curRunStatus != RunStop {
				runStatusMap[e.RunId] = RunComplete
			} else if e.Status == RunStart && curRunStatus != RunStop && curRunStatus != RunComplete {
				runStatusMap[e.RunId] = RunStart
			}
		}
	}
	return runStatusMap
}

func RunHistoryRowsToEvents(rows []map[string]any) ([]*RunHistoryEvent, error) {
	result := make([]*RunHistoryEvent, len(rows))
	for rowIdx, r := range rows {
		var err error
		result[rowIdx], err = NewRunHistoryEventFromMap(r, RunHistoryEventAllFields())
		if err != nil {
			return nil, fmt.Errorf("cannot deserialize run history row %v: %s", r, err.Error())
		}
	}
	slices.SortFunc(result, func(l, r *RunHistoryEvent) int {
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

func RunHistoryEventsToLifespanMap(events []*RunHistoryEvent) (RunLifespanMap, error) {
	runLifespanMap := RunLifespanMap{}
	emptyUnix := time.Time{}.Unix()
	for _, e := range events {
		if e.Status == RunStart {
			runLifespanMap[e.RunId] = &RunLifespan{RunId: e.RunId, StartTs: e.Ts, StartComment: e.Comment, FinalStatus: RunStart, CompletedTs: time.Time{}, StoppedTs: time.Time{}}
		} else {
			_, ok := runLifespanMap[e.RunId]
			if !ok {
				return nil, fmt.Errorf("unexpected sequence of run status events: %v", events)
			}
			if e.Status == RunComplete && runLifespanMap[e.RunId].CompletedTs.Unix() == emptyUnix {
				runLifespanMap[e.RunId].CompletedTs = e.Ts
				runLifespanMap[e.RunId].CompletedComment = e.Comment
				if runLifespanMap[e.RunId].StoppedTs.Unix() == emptyUnix {
					runLifespanMap[e.RunId].FinalStatus = RunComplete // If it was not stopped so far, consider it complete
				}
			} else if e.Status == RunStop && runLifespanMap[e.RunId].StoppedTs.Unix() == emptyUnix {
				runLifespanMap[e.RunId].StoppedTs = e.Ts
				runLifespanMap[e.RunId].StoppedComment = e.Comment
				runLifespanMap[e.RunId].FinalStatus = RunStop // Stop always wins as final status, it may be sign for dependency checker to declare results invalid (depending on the rules)
			}
		}
	}

	return runLifespanMap, nil
}

func RunHistoryRowsToStatus(rows []map[string]any) (RunStatusType, error) {
	lastStatus := RunNone
	lastTs := time.Unix(0, 0)
	fields := []string{"ts", "status"}
	for _, r := range rows {
		rec, err := NewRunHistoryEventFromMap(r, fields)
		if err != nil {
			return RunNone, fmt.Errorf("cannot deserialize status from run history row %v: %s", r, err.Error())
		}

		if rec.Ts.After(lastTs) {
			lastTs = rec.Ts
			lastStatus = RunStatusType(rec.Status)
		}
	}

	return lastStatus, nil
}
