package wfmodel

import (
	"encoding/json"
	"fmt"
)

/*
Message - carries data and signals to processors/nodes
1. No version support. Premature optimization is the root of all evil.
2. Used for data transfer only (no control signals).
3. For faster de/serialization, consider custom parser not involving reflection
4. Timestamps are int (not uint) because Unix epoch is int
*/
type Message struct {
	Ts              int64  `json:"ts"` // Assigned by sender on creation, used only for daemon statistics, see logging age
	Id              string `json:"id"` // Assigned by sender on creation, used by workers when communicating to CapiMQ/ActiveMQ and its capimq counterpart in CapimqInternalMessage - internally by CapiMQ
	ScriptURL       string `json:"script_url"`
	ScriptParamsURL string `json:"script_params_url"`
	DataKeyspace    string `json:"ks"`
	RunId           int16  `json:"run_id"`
	TargetNodeName  string `json:"target_node"`
	FirstToken      int64  `json:"first_token"`
	LastToken       int64  `json:"last_token"`
	BatchIdx        int16  `json:"batch_idx"`
	BatchesTotal    int16  `json:"batches_total"`
}

func (msg *Message) FullBatchId() string {
	return fmt.Sprintf("%s/%d/%s/%d", msg.DataKeyspace, msg.RunId, msg.TargetNodeName, msg.BatchIdx)
}

func (msg *Message) FullNodeId() string {
	return fmt.Sprintf("%s/%d/%s", msg.DataKeyspace, msg.RunId, msg.TargetNodeName)
}

func (msg *Message) ToString() string {
	return fmt.Sprintf("Ts: %d, Id:%s ScriptURL:%s,ScriptParamsURL:%s, DataKeyspace:%s, RunId:%d, TargetNodeName:%s, FirstToken:%d, LastToken:%d, BatchIdx:%d, BatchesTotal:%d. ",
		msg.Ts, msg.Id, msg.ScriptURL, msg.ScriptParamsURL, msg.DataKeyspace, msg.RunId, msg.TargetNodeName, msg.FirstToken, msg.LastToken, msg.BatchIdx, msg.BatchesTotal)
}

func (msg *Message) Deserialize(jsonBytes []byte) error {
	return json.Unmarshal(jsonBytes, msg)
}

func (msg Message) Serialize() ([]byte, error) {
	var jsonBytes []byte
	jsonBytes, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	return jsonBytes, nil
}
