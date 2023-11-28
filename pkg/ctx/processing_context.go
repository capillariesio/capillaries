package ctx

import (
	"github.com/capillariesio/capillaries/pkg/db"
	"github.com/capillariesio/capillaries/pkg/env"
	"github.com/capillariesio/capillaries/pkg/sc"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
	"github.com/gocql/gocql"
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

func (pCtx *MessageProcessingContext) DbConnect(envConfig *env.EnvConfig) error {
	var err error
	if pCtx.CqlSession, err = db.NewSession(envConfig, pCtx.BatchInfo.DataKeyspace, db.CreateKeyspaceOnConnect); err != nil {
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
