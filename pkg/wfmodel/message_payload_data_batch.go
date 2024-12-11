package wfmodel

import (
	"encoding/json"
	"fmt"
)

type MessagePayloadDataBatch struct {
	ScriptURL       string `json:"script_url"`
	ScriptParamsURL string `json:"script_params_url"`
	DataKeyspace    string `json:"data_keyspace"` // Instance/process id
	RunId           int16  `json:"run_id"`
	TargetNodeName  string `json:"target_node"`
	FirstToken      int64  `json:"first_token"`
	LastToken       int64  `json:"last_token"`
	BatchIdx        int16  `json:"batch_idx"`
	BatchesTotal    int16  `json:"batches_total"`
}

func (dc *MessagePayloadDataBatch) FullBatchId() string {
	return fmt.Sprintf("%s/%d/%s/%d", dc.DataKeyspace, dc.RunId, dc.TargetNodeName, dc.BatchIdx)
}

func (dc *MessagePayloadDataBatch) ToString() string {
	return fmt.Sprintf("ScriptURL:%s,ScriptParamsURL:%s, DataKeyspace:%s, RunId:%d, TargetNodeName:%s, FirstToken:%d, LastToken:%d, BatchIdx:%d, BatchesTotal:%d. ",
		dc.ScriptURL, dc.ScriptParamsURL, dc.DataKeyspace, dc.RunId, dc.TargetNodeName, dc.FirstToken, dc.LastToken, dc.BatchIdx, dc.BatchesTotal)
}

func (dc *MessagePayloadDataBatch) Deserialize(jsonBytes []byte) error {
	return json.Unmarshal(jsonBytes, dc)
}

func (dc MessagePayloadDataBatch) Serialize() ([]byte, error) {
	var jsonBytes []byte
	jsonBytes, err := json.Marshal(dc)
	if err != nil {
		return nil, err
	}
	return jsonBytes, nil
}
