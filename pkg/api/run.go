package api

import (
	"fmt"
	"strings"
	"time"

	"github.com/capillariesio/capillaries/pkg/cql"
	"github.com/capillariesio/capillaries/pkg/db"
	"github.com/capillariesio/capillaries/pkg/env"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/proc"
	"github.com/capillariesio/capillaries/pkg/sc"
	"github.com/capillariesio/capillaries/pkg/wf"
	"github.com/capillariesio/capillaries/pkg/wfdb"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
	"github.com/gocql/gocql"
	amqp "github.com/rabbitmq/amqp091-go"
)

func StopRun(logger *l.Logger, cqlSession *gocql.Session, keyspace string, runId int16, comment string) error {
	logger.PushF("api.StopRun")
	defer logger.PopF()

	if err := checkKeyspaceName(keyspace); err != nil {
		return err
	}

	return wfdb.SetRunStatus(logger, cqlSession, keyspace, runId, wfmodel.RunStop, comment, cql.IgnoreIfExists)
}

func StartRun(envConfig *env.EnvConfig, logger *l.Logger, amqpChannel *amqp.Channel, scriptFilePath string, paramsFilePath string, cqlSession *gocql.Session, keyspace string, startNodes []string, desc string) (int16, error) {
	logger.PushF("api.StartRun")
	defer logger.PopF()

	if err := checkKeyspaceName(keyspace); err != nil {
		return 0, err
	}

	script, err, _ := sc.NewScriptFromFiles(envConfig.CaPath, envConfig.PrivateKeys, scriptFilePath, paramsFilePath, envConfig.CustomProcessorDefFactoryInstance, envConfig.CustomProcessorsSettings)
	if err != nil {
		return 0, err
	}

	// Verify that all start nodes actually present
	missingNodesSb := strings.Builder{}
	for _, nodeName := range startNodes {
		if _, ok := script.ScriptNodes[nodeName]; !ok {
			if missingNodesSb.Len() > 0 {
				missingNodesSb.WriteString(",")
			}
			missingNodesSb.WriteString(nodeName)
		}
	}
	if missingNodesSb.Len() > 0 {
		return 0, fmt.Errorf("node(s) %s missing from %s, check node name spelling", missingNodesSb.String(), scriptFilePath)
	}

	// Get new run_id
	runId, err := wfdb.GetNextRunCounter(logger, cqlSession, keyspace)
	if err != nil {
		return 0, err
	}
	logger.Info("incremented run_id to %d", runId)

	// Write affected nodes
	affectedNodes := script.GetAffectedNodes(startNodes)
	if err := wfdb.WriteRunProperties(logger, cqlSession, keyspace, runId, startNodes, affectedNodes, scriptFilePath, paramsFilePath, desc); err != nil {
		return 0, err
	}

	logger.Info("creating data and idx tables for run %d...", runId)

	// Create all run-specific tables, do not create them in daemon on the fly to avoid INCOMPATIBLE_SCHEMA error
	// (apparently, thrown if we try to insert immediately after creating a table)
	tablesCreated := 0
	for _, nodeName := range affectedNodes {
		node, ok := script.ScriptNodes[nodeName]
		if !ok || !node.HasTableCreator() {
			continue
		}
		q := proc.CreateDataTableCql(keyspace, runId, &node.TableCreator)
		if err := cqlSession.Query(q).Exec(); err != nil {
			return 0, db.WrapDbErrorWithQuery("cannot create data table", q, err)
		}
		tablesCreated++
		for idxName, idxDef := range node.TableCreator.Indexes {
			q = proc.CreateIdxTableCql(keyspace, runId, idxName, idxDef)
			if err := cqlSession.Query(q).Exec(); err != nil {
				return 0, db.WrapDbErrorWithQuery("cannot create idx table", q, err)
			}
			tablesCreated++
		}
	}

	logger.Info("created %d tables, creating messages to send for run %d...", tablesCreated, runId)

	allMsgs := make([]*wfmodel.Message, 0)
	allHandlerExeTypes := make([]string, 0)
	for _, affectedNodeName := range affectedNodes {
		affectedNode, ok := script.ScriptNodes[affectedNodeName]
		if !ok {
			return 0, fmt.Errorf("cannot find node to start with: %s in the script %s", affectedNodeName, scriptFilePath)
		}
		intervals, err := affectedNode.GetTokenIntervalsByNumberOfBatches()
		if err != nil {
			return 0, err
		}
		msgs := make([]*wfmodel.Message, len(intervals))
		handlerExeTypes := make([]string, len(intervals))
		for msgIdx := 0; msgIdx < len(intervals); msgIdx++ {
			msgs[msgIdx] = &wfmodel.Message{
				Ts:          time.Now().UnixMilli(),
				MessageType: wfmodel.MessageTypeDataBatch,
				Payload: wfmodel.MessagePayloadDataBatch{
					ScriptURI:       scriptFilePath,
					ScriptParamsURI: paramsFilePath,
					DataKeyspace:    keyspace,
					RunId:           runId,
					TargetNodeName:  affectedNodeName,
					FirstToken:      intervals[msgIdx][0],
					LastToken:       intervals[msgIdx][1],
					BatchIdx:        int16(msgIdx),
					BatchesTotal:    int16(len(intervals))}}
			handlerExeTypes[msgIdx] = affectedNode.HandlerExeType
		}
		allMsgs = append(allMsgs, msgs...)
		allHandlerExeTypes = append(allHandlerExeTypes, handlerExeTypes...)
	}

	// Write status 'start', fail if a record for run_id is already there (too many operators)
	if err := wfdb.SetRunStatus(logger, cqlSession, keyspace, runId, wfmodel.RunStart, "api.StartRun", cql.ThrowIfExists); err != nil {
		return 0, err
	}

	logger.Info("sending %d messages for run %d...", len(allMsgs), runId)

	// Send one msg after another
	// TODO: there easily may be hundreds of messages, can we send them in a single shot?
	for msgIdx := 0; msgIdx < len(allMsgs); msgIdx++ {
		msgOutBytes, errMsgOut := allMsgs[msgIdx].Serialize()
		if errMsgOut != nil {
			return 0, fmt.Errorf("cannot serialize outgoing message %v. %v", allMsgs[msgIdx].ToString(), errMsgOut)
		}

		errSend := amqpChannel.Publish(
			envConfig.Amqp.Exchange,    // exchange
			allHandlerExeTypes[msgIdx], // routing key / hander exe type
			false,                      // mandatory
			false,                      // immediate
			amqp.Publishing{ContentType: "text/plain", Body: msgOutBytes})
		if errSend != nil {
			// Reconnect required
			return 0, fmt.Errorf("failed to send next message: %v\n", errSend)
		}
	}
	return runId, nil
}

func RunNode(envConfig *env.EnvConfig, logger *l.Logger, nodeName string, runId int16, scriptFilePath string, paramsFilePath string, cqlSession *gocql.Session, keyspace string) (int16, error) {
	logger.PushF("api.RunNode")
	defer logger.PopF()

	script, err, _ := sc.NewScriptFromFiles(envConfig.CaPath, envConfig.PrivateKeys, scriptFilePath, paramsFilePath, envConfig.CustomProcessorDefFactoryInstance, envConfig.CustomProcessorsSettings)
	if err != nil {
		return 0, err
	}
	// Get new run_id if needed
	if runId == 0 {
		runId, err = wfdb.GetNextRunCounter(logger, cqlSession, keyspace)
		if err != nil {
			return 0, err
		}
		logger.Info("incremented run_id to %d", runId)
	}

	// Calculate intervals for this node
	node, ok := script.ScriptNodes[nodeName]
	if !ok {
		return 0, fmt.Errorf("cannot find node to start with: %s in the script %s", nodeName, scriptFilePath)
	}

	intervals, err := node.GetTokenIntervalsByNumberOfBatches()
	if err != nil {
		return 0, err
	}

	// Write affected nodes
	affectedNodes := script.GetAffectedNodes([]string{nodeName})
	if err := wfdb.WriteRunProperties(logger, cqlSession, keyspace, runId, []string{nodeName}, affectedNodes, scriptFilePath, paramsFilePath, "started by Toolbelt direct RunNode"); err != nil {
		return 0, err
	}

	// Write status 'start', fail if a record for run_id is already there (too many operators)
	if err := wfdb.SetRunStatus(logger, cqlSession, keyspace, runId, wfmodel.RunStart, fmt.Sprintf("Toolbelt RunNode(%s)", nodeName), cql.ThrowIfExists); err != nil {
		return 0, err
	}

	logger.Info("creating data and idx tables for run %d...", runId)

	// Create all run-specific tables, do not create them in daemon on the fly to avoid INCOMPATIBLE_SCHEMA error
	// (apparently, thrown if we try to insert immediately after creating a table)
	tablesCreated := 0
	for _, nodeName := range affectedNodes {
		node, ok := script.ScriptNodes[nodeName]
		if !ok || !node.HasTableCreator() {
			continue
		}
		q := proc.CreateDataTableCql(keyspace, runId, &node.TableCreator)
		if err := cqlSession.Query(q).Exec(); err != nil {
			return 0, db.WrapDbErrorWithQuery("cannot create data table", q, err)
		}
		tablesCreated++
		for idxName, idxDef := range node.TableCreator.Indexes {
			q = proc.CreateIdxTableCql(keyspace, runId, idxName, idxDef)
			if err := cqlSession.Query(q).Exec(); err != nil {
				return 0, db.WrapDbErrorWithQuery("cannot create idx table", q, err)
			}
			tablesCreated++
		}
	}

	logger.Info("created %d tables, creating messages to send for run %d...", tablesCreated, runId)

	for i := 0; i < len(intervals); i++ {
		batchStartTs := time.Now()
		logger.Info("BatchStarted: [%d,%d]...", intervals[i][0], intervals[i][1])
		dataBatchInfo := wfmodel.MessagePayloadDataBatch{
			ScriptURI:       scriptFilePath,
			ScriptParamsURI: paramsFilePath,
			DataKeyspace:    keyspace,
			RunId:           runId,
			TargetNodeName:  nodeName,
			FirstToken:      intervals[i][0],
			LastToken:       intervals[i][1],
			BatchIdx:        int16(i),
			BatchesTotal:    int16(len(intervals))}

		if daemonCmd := wf.ProcessDataBatchMsg(envConfig, logger, batchStartTs.UnixMilli(), &dataBatchInfo); daemonCmd != wf.DaemonCmdAckSuccess {
			return 0, fmt.Errorf("processor returned daemon cmd %d, assuming failure, check the logs", daemonCmd)
		}
		logger.Info("BatchComplete: [%d,%d], %.3fs", intervals[i][0], intervals[i][1], time.Since(batchStartTs).Seconds())
	}
	if err := wfdb.SetRunStatus(logger, cqlSession, keyspace, runId, wfmodel.RunComplete, fmt.Sprintf("Toolbelt RunNode(%s), run successful", nodeName), cql.IgnoreIfExists); err != nil {
		return 0, err
	}

	return runId, nil
}
