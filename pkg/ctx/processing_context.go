package ctx

import (
	"fmt"
	"time"

	"github.com/capillariesio/capillaries/pkg/cql"
	"github.com/capillariesio/capillaries/pkg/env"
	"github.com/capillariesio/capillaries/pkg/sc"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
	"github.com/gocql/gocql"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type MessageProcessingContext struct {
	MsgTs             int64
	BatchInfo         wfmodel.MessagePayloadDataBatch
	CqlSession        *gocql.Session
	Script            *sc.ScriptDef
	CurrentScriptNode *sc.ScriptNodeDef
	ZapDataKeyspace   zapcore.Field
	ZapRun            zapcore.Field
	ZapNode           zapcore.Field
	ZapBatchIdx       zapcore.Field
	ZapMsgAgeMillis   zapcore.Field
}

func NewFromBatchInfo(envConfig *env.EnvConfig, msgTs int64, batchInfo *wfmodel.MessagePayloadDataBatch) (*MessageProcessingContext, error) {
	pCtx := &MessageProcessingContext{
		MsgTs:           msgTs,
		BatchInfo:       *batchInfo,
		ZapDataKeyspace: zap.String("ks", batchInfo.DataKeyspace),
		ZapRun:          zap.Int16("run", batchInfo.RunId),
		ZapNode:         zap.String("node", batchInfo.TargetNodeName),
		ZapBatchIdx:     zap.Int16("bi", batchInfo.BatchIdx),
		ZapMsgAgeMillis: zap.Int64("age", time.Now().UnixMilli()-msgTs)}

	var err error
	pCtx.Script, err = sc.NewScriptFromFiles(envConfig.CaPath, envConfig.PrivateKeys, batchInfo.ScriptURI, batchInfo.ScriptParamsURI, envConfig.CustomProcessorDefFactoryInstance, envConfig.CustomProcessorsSettings)
	if err != nil {
		return nil, fmt.Errorf("cannot initialize context with script, giving up with msg %s returning DaemonCmdAckWithError: %s", batchInfo.ToString(), err.Error())
	}

	var ok bool
	pCtx.CurrentScriptNode, ok = pCtx.Script.ScriptNodes[batchInfo.TargetNodeName]
	if !ok {
		return nil, fmt.Errorf("cannot find node %s in the script [%s], giving up with %s, returning DaemonCmdAckWithError", pCtx.BatchInfo.TargetNodeName, pCtx.BatchInfo.ScriptURI, batchInfo.ToString())
	}

	return pCtx, nil
}

func (pCtx *MessageProcessingContext) DbConnect(envConfig *env.EnvConfig) error {
	var err error
	if pCtx.CqlSession, err = cql.NewSession(envConfig, pCtx.BatchInfo.DataKeyspace, cql.CreateKeyspaceOnConnect); err != nil {
		return err
	}
	return nil
}

func (pCtx *MessageProcessingContext) DbClose() {
	if pCtx.CqlSession != nil {
		if pCtx.CqlSession.Closed() {
			// TODO: something is not clean in the code, find a way to communicate it without using logger
		} else {
			pCtx.CqlSession.Close()
		}
	}
}

// func (pCtx *MessageProcessingContext) InitScript(envConfig *env.EnvConfig) error {

// 	var err error
// 	pCtx.Script, err = sc.NewScriptFromFiles(envConfig.CaPath, envConfig.PrivateKeys, pCtx.BatchInfo.ScriptURI, pCtx.BatchInfo.ScriptParamsURI, envConfig.CustomProcessorDefFactoryInstance, envConfig.CustomProcessorsSettings)
// 	if err != nil {
// 		return fmt.Errorf("cannot initialize context with script: %s", err.Error())
// 	}

// 	var ok bool
// 	pCtx.CurrentScriptNode, ok = pCtx.Script.ScriptNodes[pCtx.BatchInfo.TargetNodeName]
// 	if !ok {
// 		return fmt.Errorf("cannot find node %s in the script [%s]", pCtx.BatchInfo.TargetNodeName, pCtx.BatchInfo.ScriptURI)
// 	}

// 	return nil
// }
