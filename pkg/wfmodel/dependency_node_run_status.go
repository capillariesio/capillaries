package wfmodel

import (
	"fmt"
	"strings"

	"github.com/capillariesio/capillaries/pkg/eval"
)

// Fictional table, used by dependency checker, look for this name in Capillaries scripts with dependency check strategy json/yaml
const DependencyNodeRunStatusTableName string = "nrs"

// Used by dependency checker: nrs.run_is_current == true && nrs.run_status == wfmodel.RunStart && nrs.node_status
type DependencyNodeRunStatus struct {
	RunId        int16
	RunIsCurrent bool
	RunStatus    RunStatusType
	NodeStatus   NodeBatchStatusType
	SortKey      string
}

// Used for logging only
func (e *DependencyNodeRunStatus) ToString() string {
	sb := strings.Builder{}
	sb.WriteString("{")
	fmt.Fprintf(&sb, "run_id:%d,", e.RunId)
	fmt.Fprintf(&sb, "run_is_current:%t,", e.RunIsCurrent)
	fmt.Fprintf(&sb, "run_status:%s,", e.RunStatus.ToString())
	fmt.Fprintf(&sb, "node_status:%s,", NodeBatchStatusToString(e.NodeStatus))
	sb.WriteString("}")
	return sb.String()
}

// This slice is passed to CheckDependencyPolicyAgainstNodeEventList
type DependencyNodeRunStatusSlice []DependencyNodeRunStatus

// Used for logging only
func (statuses DependencyNodeRunStatusSlice) ToString() string {
	items := make([]string, len(statuses))
	for eventIdx := 0; eventIdx < len(statuses); eventIdx++ {
		items[eventIdx] = statuses[eventIdx].ToString()
	}
	return fmt.Sprintf("[%s]", strings.Join(items, ", "))
}

// From DependencyNodeRunStatus struct to a db-style map that can be used by expression evaluator in dependency checker
func DependencyCheckerNodeVars(nrs DependencyNodeRunStatus) eval.VarValuesMap {
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
		"run_id":         int64(nrs.RunId),
		"run_is_current": nrs.RunIsCurrent,
		"run_status":     int64(nrs.RunStatus),
		"node_status":    int64(nrs.NodeStatus),
	}
	return m
}
