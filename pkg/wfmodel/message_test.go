package wfmodel

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMessage(t *testing.T) {
	m := Message{
		Ts:              int64(64),
		Id:              "someid",
		ScriptURL:       "scripturl",
		ScriptParamsURL: "scriptparamsurl",
		DataKeyspace:    "ks",
		RunId:           int16(16),
		TargetNodeName:  "targetnode",
		FirstToken:      int64(641),
		LastToken:       int64(649),
		BatchIdx:        int16(161),
		BatchesTotal:    int16(169),
	}

	assert.Equal(t, "ks/16/targetnode/161", m.FullBatchId())
	assert.Equal(t, "ks/16/targetnode", m.FullNodeId())
	assert.Equal(t, "Ts: 64, Id:someid ScriptURL:scripturl,ScriptParamsURL:scriptparamsurl, DataKeyspace:ks, RunId:16, TargetNodeName:targetnode, FirstToken:641, LastToken:649, BatchIdx:161, BatchesTotal:169. ", m.ToString())

	b, err := m.Serialize()
	assert.Nil(t, err)

	m2 := Message{}
	err = m2.Deserialize(b)
	assert.Nil(t, err)
	assert.Equal(t, m.ToString(), m2.ToString())
}
