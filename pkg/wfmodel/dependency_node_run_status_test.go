package wfmodel

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// No functionality tested, just to raise concerns when some constants change
func TestDependencyCheckerNodeVars(t *testing.T) {
	nrs := DependencyNodeRunStatus{
		RunId:        int16(16),
		RunIsCurrent: true,
		RunStatus:    RunStart,
		NodeStatus:   NodeBatchStart,
	}
	vars := DependencyCheckerNodeVars(nrs)

	assert.Equal(t, int64(NodeBatchNone), vars[WfmodelNamespace]["NodeBatchNone"])
	assert.Equal(t, int64(NodeBatchStart), vars[WfmodelNamespace]["NodeBatchStart"])
	assert.Equal(t, int64(NodeBatchSuccess), vars[WfmodelNamespace]["NodeBatchSuccess"])
	assert.Equal(t, int64(NodeBatchFail), vars[WfmodelNamespace]["NodeBatchFail"])
	assert.Equal(t, int64(NodeBatchRunStopReceived), vars[WfmodelNamespace]["NodeBatchRunStopReceived"])

	assert.Equal(t, int64(RunNone), vars[WfmodelNamespace]["RunNone"])
	assert.Equal(t, int64(RunStart), vars[WfmodelNamespace]["RunStart"])
	assert.Equal(t, int64(RunComplete), vars[WfmodelNamespace]["RunComplete"])
	assert.Equal(t, int64(RunStop), vars[WfmodelNamespace]["RunStop"])

	assert.Equal(t, int64(nrs.RunId), vars[DependencyNodeRunStatusTableName]["run_id"])
	assert.Equal(t, nrs.RunIsCurrent, vars[DependencyNodeRunStatusTableName]["run_is_current"])
	assert.Equal(t, int64(nrs.RunStatus), vars[DependencyNodeRunStatusTableName]["run_status"])
	assert.Equal(t, int64(nrs.NodeStatus), vars[DependencyNodeRunStatusTableName]["node_status"])
}

func TestLoggingHelpers(t *testing.T) {
	nrsSlice := DependencyNodeRunStatusSlice{
		DependencyNodeRunStatus{
			RunId:        int16(16),
			RunIsCurrent: true,
			RunStatus:    RunStart,
			NodeStatus:   NodeBatchStart,
		}}
	assert.Equal(t, "[{run_id:16,run_is_current:true,run_status:start,node_status:start,}]", nrsSlice.ToString())
}
