package wfmodel

import (
	"fmt"
	"strings"

	"github.com/capillariesio/capillaries/pkg/eval"
)

// Fictional table, used by dependency checker
const DependencyNodeRunStatusTableName string = "nrs"

// nrs.run_is_current == true && nrs.run_status == wfmodel.RunStart && nrs.node_status
type DependencyNodeRunStatus struct {
	RunId        int16
	RunIsCurrent bool
	RunStatus    RunStatusType
	NodeStatus   NodeBatchStatusType
	SortKey      string
}

func (e *DependencyNodeRunStatus) ToVars() eval.VarValuesMap {
	return eval.VarValuesMap{
		DependencyNodeRunStatusTableName: map[string]any{
			"run_id":         int64(e.RunId),
			"run_is_current": e.RunIsCurrent,
			"run_status":     int64(e.RunStatus),
			"node_status":    int64(e.NodeStatus),
		}}
}

func (e *DependencyNodeRunStatus) ToString() string {
	sb := strings.Builder{}
	sb.WriteString("{")
	fmt.Fprintf(&sb, "run_id:%d,", e.RunId)
	fmt.Fprintf(&sb, "run_is_current:%t,", e.RunIsCurrent)
	fmt.Fprintf(&sb, "run_status:%s,", e.RunStatus.ToString())
	fmt.Fprintf(&sb, "node_status:%s,", e.NodeStatus.ToString())
	sb.WriteString("}")
	return sb.String()
}

type DependencyNodeRunStatusSlice []DependencyNodeRunStatus

func (statuses DependencyNodeRunStatusSlice) ToString() string {
	items := make([]string, len(statuses))
	for eventIdx := 0; eventIdx < len(statuses); eventIdx++ {
		items[eventIdx] = statuses[eventIdx].ToString()
	}
	return fmt.Sprintf("[%s]", strings.Join(items, ", "))
}

func NewVarsFromDepCtx(e DependencyNodeRunStatus) eval.VarValuesMap {
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
	m[DependencyNodeRunStatusTableName] = map[string]any{
		"run_id":         int64(e.RunId),
		"run_is_current": e.RunIsCurrent,
		"run_status":     int64(e.RunStatus),
		"node_status":    int64(e.NodeStatus),
	}
	return m
}
