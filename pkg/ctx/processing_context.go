package ctx

import (
	"fmt"
	"time"

	"github.com/gocql/gocql"
	"github.com/kleineshertz/capillaries/pkg/cql"
	"github.com/kleineshertz/capillaries/pkg/env"
	"github.com/kleineshertz/capillaries/pkg/sc"
	"github.com/kleineshertz/capillaries/pkg/wfmodel"
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

func NewFromBatchInfo(msgTs int64, batchInfo *wfmodel.MessagePayloadDataBatch) *MessageProcessingContext {
	return &MessageProcessingContext{
		MsgTs:           msgTs,
		BatchInfo:       *batchInfo,
		ZapDataKeyspace: zap.String("ks", batchInfo.DataKeyspace),
		ZapRun:          zap.Int16("run", batchInfo.RunId),
		ZapNode:         zap.String("node", batchInfo.TargetNodeName),
		ZapBatchIdx:     zap.Int16("bi", batchInfo.BatchIdx),
		ZapMsgAgeMillis: zap.Int64("age", time.Now().UnixMilli()-msgTs)}
}

func (pCtx *MessageProcessingContext) DbConnect(envConfig *env.EnvConfig) error {
	var err error
	if pCtx.CqlSession, err = cql.NewSession(envConfig, pCtx.BatchInfo.DataKeyspace); err != nil {
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

func (pCtx *MessageProcessingContext) InitScript(envConfig *env.EnvConfig) error {

	var err error
	pCtx.Script, err = sc.NewScriptFromFiles(envConfig.CaPath, pCtx.BatchInfo.ScriptURI, pCtx.BatchInfo.ScriptParamsURI, envConfig.CustomProcessorDefFactoryInstance, envConfig.CustomProcessorsSettings)
	if err != nil {
		return fmt.Errorf("cannot initialize context with script: %s", err.Error())
	}

	var ok bool
	pCtx.CurrentScriptNode, ok = pCtx.Script.ScriptNodes[pCtx.BatchInfo.TargetNodeName]
	if !ok {
		return fmt.Errorf("cannot find node %s in the script [%s]", pCtx.BatchInfo.TargetNodeName, pCtx.BatchInfo.ScriptURI)
	}

	return nil
}
