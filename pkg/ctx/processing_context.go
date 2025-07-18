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
	CassandraEngine   db.CassandraEngineType
}

func (pCtx *MessageProcessingContext) DbConnect(envConfig *env.EnvConfig) error {
	var err error
	if pCtx.CqlSession, pCtx.CassandraEngine, err = db.NewSession(envConfig, pCtx.BatchInfo.DataKeyspace, db.DoNotCreateKeyspaceOnConnect); err != nil {
		return err
	}
	// rnd := rand.New(rand.NewSource(time.Now().UnixMilli()))
	// if rnd.Float32() > 0.60 {
	// 	return fmt.Errorf("random db error for test")
	// }
	return nil
}

func (pCtx *MessageProcessingContext) DbClose() {
	if pCtx.CqlSession != nil {
		// TODO: if it's already closed, something is not clean in the code, find a way to communicate it without using logger
		if !pCtx.CqlSession.Closed() {
			pCtx.CqlSession.Close()
		}
	}
}
