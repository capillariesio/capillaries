package ctx

import (
	"time"

	"github.com/capillariesio/capillaries/pkg/db"
	"github.com/capillariesio/capillaries/pkg/env"
	"github.com/capillariesio/capillaries/pkg/sc"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
	"github.com/gocql/gocql"
	"go.uber.org/zap/zapcore"
)

type HeartbeatCallbackFunc func(string)

type MessageProcessingContext struct {
	Msg                     wfmodel.Message
	CqlSession              *gocql.Session
	Script                  *sc.ScriptDef
	CurrentScriptNode       *sc.ScriptNodeDef
	ZapMsgId                zapcore.Field
	ZapDataKeyspace         zapcore.Field
	ZapRun                  zapcore.Field
	ZapNode                 zapcore.Field
	ZapBatchIdx             zapcore.Field
	ZapMsgAgeMillis         zapcore.Field
	CassandraEngine         db.CassandraEngineType
	LastHeartbeatSentTs     int64
	HeartbeatIntervalMillis int64
	HeartbeatCallback       HeartbeatCallbackFunc
}

func (pCtx *MessageProcessingContext) DbConnect(envConfig *env.EnvConfig) error {
	var err error
	if pCtx.CqlSession, pCtx.CassandraEngine, err = db.NewSession(envConfig, pCtx.Msg.DataKeyspace, db.DoNotCreateKeyspaceOnConnect); err != nil {
		return err
	}
	// rnd := rand.New(rand.NewSource(time.Now().UnixMilli()))
	// if rnd.Float32() > 0.60 {
	// 	return fmt.Errorf("random db error for test")
	// }
	return nil
}

func (pCtx *MessageProcessingContext) SendHeartbeat() {
	if pCtx.HeartbeatCallback != nil && pCtx.HeartbeatIntervalMillis > 0 {
		now := time.Now().UnixMilli()
		if pCtx.LastHeartbeatSentTs+pCtx.HeartbeatIntervalMillis < now {
			pCtx.LastHeartbeatSentTs = now
			pCtx.HeartbeatCallback(pCtx.Msg.Id)
		}
	}
}

func (pCtx *MessageProcessingContext) DbClose() {
	if pCtx.CqlSession != nil {
		// TODO: if it's already closed, something is not clean in the code, find a way to communicate it without using logger
		if !pCtx.CqlSession.Closed() {
			pCtx.CqlSession.Close()
		}
	}
}
