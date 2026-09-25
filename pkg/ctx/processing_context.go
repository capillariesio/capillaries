package ctx

import (
	"time"

	"github.com/capillariesio/capillaries/pkg/db"
	"github.com/capillariesio/capillaries/pkg/env"
	"github.com/capillariesio/capillaries/pkg/sc"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
	"github.com/capillariesio/gocqlmem/gocqlshims"
	"go.uber.org/zap/zapcore"
)

type HeartbeatCallbackFunc func(string)

type TestScenarioType int

// const (
// 	TestProduction TestScenarioType = iota
// 	// Data table: not exist, timeout, serious, not applied
// 	TestDataDoesNotExist
// 	TestDataOperationTimedOut
// 	TestDataSerious
// 	TestDataNotApplied
// 	// Idx table: not exist, timeout, serious, not applied
// 	TestIdxDoesNotExist
// 	TestIdxOperationTimedOut
// 	TestIdxSerious
// 	TestIdxNotAppliedSamePresentFirstRun
// 	TestIdxNotAppliedSamePresentSecondRun
// 	TestIdxNotAppliedDiffPresent
// 	// Generic process batch error
// 	TestProcessDataBatchError
// )

type TableInserterProperties struct {
	QueryPerformer               db.TableInserterQueryPerformer
	DoesNotExistPauseMillis      int64
	OperationTimedOutPauseMillis int64
}

type MessageProcessingContext struct {
	Msg                     wfmodel.Message
	CqlSession              gocqlshims.Session
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
	// TestScenario            TestScenarioType
	TableInserterProps TableInserterProperties
}

func CreateProductionTableInserterProperties() TableInserterProperties {
	return TableInserterProperties{
		QueryPerformer:               &db.TableInserterQueryPerformerProduction{},
		DoesNotExistPauseMillis:      2000, // 2000, 5 retries: 2000 + 4000 + 8000 + 16000 + 32000
		OperationTimedOutPauseMillis: 200,  // 200, 5 retries: 200 + 400 + 800 + 1600 + 3200 = 6200
	}
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
