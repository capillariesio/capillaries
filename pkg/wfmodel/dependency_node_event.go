package wfmodel

import (
	"fmt"
	"strings"
	"time"

	"github.com/capillariesio/capillaries/pkg/eval"
)

const DependencyNodeEventTableName string = "e"

type DependencyNodeEvent struct {
	RunId          int16
	RunIsCurrent   bool
	RunStartTs     time.Time
	RunFinalStatus RunStatusType
	RunCompletedTs time.Time
	RunStoppedTs   time.Time
	NodeIsStarted  bool
	NodeStartTs    time.Time
	NodeStatus     NodeBatchStatusType
	NodeStatusTs   time.Time
	SortKey        string
}

func (e *DependencyNodeEvent) ToVars() eval.VarValuesMap {
	return eval.VarValuesMap{
		DependencyNodeEventTableName: map[string]any{
			"run_id":           int64(e.RunId),
			"run_is_current":   e.RunIsCurrent,
			"run_start_ts":     e.RunStartTs,
			"run_final_status": int64(e.RunFinalStatus),
			"run_completed_ts": e.RunCompletedTs,
			"run_stopped_ts":   e.RunStoppedTs,
			"node_is_started":  e.NodeIsStarted,
			"node_start_ts":    e.NodeStartTs,
			"node_status":      int64(e.NodeStatus),
			"node_status_ts":   e.NodeStatusTs}}
}

func (e *DependencyNodeEvent) ToString() string {
	sb := strings.Builder{}
	sb.WriteString("{")
	sb.WriteString(fmt.Sprintf("run_id:%d,", e.RunId))
	sb.WriteString(fmt.Sprintf("run_is_current:%t,", e.RunIsCurrent))
	sb.WriteString(fmt.Sprintf("run_start_ts:%s,", e.RunStartTs.Format(LogTsFormatQuoted)))
	sb.WriteString(fmt.Sprintf("run_final_status:%s,", e.RunFinalStatus.ToString()))
	sb.WriteString(fmt.Sprintf("run_completed_ts:%s,", e.RunCompletedTs.Format(LogTsFormatQuoted)))
	sb.WriteString(fmt.Sprintf("run_stopped_ts:%s,", e.RunStoppedTs.Format(LogTsFormatQuoted)))
	sb.WriteString(fmt.Sprintf("node_is_started:%t,", e.NodeIsStarted))
	sb.WriteString(fmt.Sprintf("node_start_ts:%s,", e.NodeStartTs.Format(LogTsFormatQuoted)))
	sb.WriteString(fmt.Sprintf("node_status:%s,", e.NodeStatus.ToString()))
	sb.WriteString(fmt.Sprintf("node_status_ts:%s", e.NodeStatusTs.Format(LogTsFormatQuoted)))
	sb.WriteString("}")
	return sb.String()
}

type DependencyNodeEvents []DependencyNodeEvent

func (events DependencyNodeEvents) ToString() string {
	items := make([]string, len(events))
	for eventIdx := 0; eventIdx < len(events); eventIdx++ {
		items[eventIdx] = events[eventIdx].ToString()
	}
	return fmt.Sprintf("[%s]", strings.Join(items, ", "))
}

func NewVarsFromDepCtx(e DependencyNodeEvent) eval.VarValuesMap {
	m := eval.VarValuesMap{}
	m[WfmodelNamespace] = map[string]any{
		"NodeBatchNone":            int64(NodeBatchNone),
		"NodeBatchStart":           int64(NodeBatchStart),
		"NodeBatchSuccess":         int64(NodeBatchSuccess),
		"NodeBatchFail":            int64(NodeBatchFail),
		"NodeBatchRunStopReceived": int64(NodeBatchRunStopReceived),
		"RunNone":                  int64(RunNone),
		"RunStart":                 int64(RunStart),
		"RunComplete":              int64(RunComplete),
		"RunStop":                  int64(RunStop)}
	m[DependencyNodeEventTableName] = map[string]any{
		"run_id":           int64(e.RunId),
		"run_is_current":   e.RunIsCurrent,
		"run_start_ts":     e.RunStartTs,
		"run_final_status": int64(e.RunFinalStatus),
		"run_completed_ts": e.RunCompletedTs,
		"run_stopped_ts":   e.RunStoppedTs,
		"node_is_started":  e.NodeIsStarted,
		"node_start_ts":    e.NodeStartTs,
		"node_status":      int64(e.NodeStatus),
		"node_status_ts":   e.NodeStatusTs}
	return m
}
