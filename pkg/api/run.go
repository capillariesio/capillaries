package api

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/capillariesio/capillaries/pkg/cql"
	"github.com/capillariesio/capillaries/pkg/db"
	"github.com/capillariesio/capillaries/pkg/env"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/mq"
	"github.com/capillariesio/capillaries/pkg/proc"
	"github.com/capillariesio/capillaries/pkg/sc"
	"github.com/capillariesio/capillaries/pkg/wf"
	"github.com/capillariesio/capillaries/pkg/wfdb"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
	"github.com/gocql/gocql"
	amqp091 "github.com/rabbitmq/amqp091-go"
)

// Used by Webapi and Toolbelt (stop_run command)
func StopRun(logger *l.CapiLogger, cqlSession *gocql.Session, keyspace string, runId int16, comment string) error {
	logger.PushF("api.StopRun")
	defer logger.PopF()

	if err := checkKeyspaceName(keyspace); err != nil {
		return err
	}

	return wfdb.SetRunStatus(logger, cqlSession, keyspace, runId, wfmodel.RunStop, comment, cql.IgnoreIfExists)
}

// Used by Webapi and Toolbelt (start_run command). This is the way to start Capillaries processing.
// startNodes parameter contains names of the script nodes to be executed right upon run start.
func StartRun(envConfig *env.EnvConfig, logger *l.CapiLogger, amqpChannel *amqp091.Channel, mqSender mq.MqProducer, scriptFilePath string, paramsFilePath string, cqlSession *gocql.Session, cassandraEngine db.CassandraEngineType, keyspace string, startNodes []string, desc string) (int16, error) {
	logger.PushF("api.StartRun")
	defer logger.PopF()

	if err := checkKeyspaceName(keyspace); err != nil {
		return 0, err
	}

	script, _, err := sc.NewScriptFromFiles(envConfig.CaPath, envConfig.PrivateKeys, scriptFilePath, paramsFilePath, envConfig.CustomProcessorDefFactoryInstance, envConfig.CustomProcessorsSettings)
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
	if err := wfdb.WriteRunProperties(cqlSession, keyspace, runId, startNodes, affectedNodes, scriptFilePath, paramsFilePath, desc); err != nil {
		return 0, err
	}

	logger.Info("creating data and idx tables for run %d...", runId)

	// Create all run-specific tables, do not create them in daemon on the fly to avoid INCOMPATIBLE_SCHEMA error
	// (apparently, thrown if we try to insert immediately after creating a table)
	createTablesStartTime := time.Now()
	var tableNames []string
	for _, nodeName := range affectedNodes {
		node, ok := script.ScriptNodes[nodeName]
		if !ok || !node.HasTableCreator() {
			continue
		}
		q := proc.CreateDataTableCql(keyspace, runId, &node.TableCreator)
		if err := cqlSession.Query(q).Exec(); err != nil {
			return 0, db.WrapDbErrorWithQuery("cannot create data table", q, err)
		}
		tableNames = append(tableNames, fmt.Sprintf("%s%s", node.TableCreator.Name, cql.RunIdSuffix(runId)))

		for idxName, idxDef := range node.TableCreator.Indexes {
			q = proc.CreateIdxTableCql(keyspace, runId, idxName, idxDef, &node.TableCreator)
			if err := cqlSession.Query(q).Exec(); err != nil {
				return 0, db.WrapDbErrorWithQuery("cannot create idx table", q, err)
			}
			tableNames = append(tableNames, fmt.Sprintf("%s%s", idxName, cql.RunIdSuffix(runId)))
		}
	}

	if cassandraEngine == db.CassandraEngineAmazonKeyspaces {
		if err := db.VerifyAmazonKeyspacesTablesReady(cqlSession, keyspace, tableNames); err != nil {
			return 0, err
		}
	}

	logger.Info("created %d tables [%s] in %.2fs, creating messages to send for run %d...", len(tableNames), strings.Join(tableNames, ","), time.Since(createTablesStartTime).Seconds(), runId)

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
				Id:              uuid.NewString(),
				Ts:              time.Now().UnixMilli(),
				ScriptURL:       scriptFilePath,
				ScriptParamsURL: paramsFilePath,
				DataKeyspace:    keyspace,
				RunId:           runId,
				TargetNodeName:  affectedNodeName,
				FirstToken:      intervals[msgIdx][0],
				LastToken:       intervals[msgIdx][1],
				BatchIdx:        int16(msgIdx),
				BatchesTotal:    int16(len(intervals))}
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
	sendMsgStartTime := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Send one msg after another
	// TODO: there easily may be hundreds of messages, can we send them in a single shot?
	for msgIdx := 0; msgIdx < len(allMsgs); msgIdx++ {
		// msgOutBytes, errMsgOut := allMsgs[msgIdx].Serialize()
		// if errMsgOut != nil {
		// 	return 0, fmt.Errorf("cannot serialize outgoing message %d %v. %v", msgIdx, allMsgs[msgIdx].ToString(), errMsgOut)
		// }

		// errSend := amqpChannel.PublishWithContext(
		// 	ctx,
		// 	envConfig.Amqp.Exchange,    // exchange
		// 	allHandlerExeTypes[msgIdx], // routing key / hander exe type
		// 	false,                      // mandatory
		// 	false,                      // immediate
		// 	amqp.Publishing{ContentType: "text/plain", Body: msgOutBytes})
		// if errSend != nil {
		// 	// Reconnect required
		// 	return 0, fmt.Errorf("failed to send next message %d: %s", msgIdx, errSend.Error())
		// }
		if errSend := mqSender.Send(ctx, allMsgs[msgIdx]); errSend != nil {
			return 0, fmt.Errorf("failed to send next message %d: %s", msgIdx, errSend.Error())
		}
	}
	logger.Info("sent %d msgs in %.2fs for run %d", len(allMsgs), time.Since(sendMsgStartTime).Seconds(), runId)

	return runId, nil
}

// Used by Toolbelt (exec_node command), can be used for testing Capillaries node execution without RabbitMQ wokflow
func RunNode(envConfig *env.EnvConfig, logger *l.CapiLogger, nodeName string, runId int16, scriptFilePath string, paramsFilePath string, cqlSession *gocql.Session, keyspace string) (int16, error) {
	logger.PushF("api.RunNode")
	defer logger.PopF()

	script, _, err := sc.NewScriptFromFiles(envConfig.CaPath, envConfig.PrivateKeys, scriptFilePath, paramsFilePath, envConfig.CustomProcessorDefFactoryInstance, envConfig.CustomProcessorsSettings)
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
	if err := wfdb.WriteRunProperties(cqlSession, keyspace, runId, []string{nodeName}, affectedNodes, scriptFilePath, paramsFilePath, "started by Toolbelt direct RunNode"); err != nil {
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
			q = proc.CreateIdxTableCql(keyspace, runId, idxName, idxDef, &node.TableCreator)
			if err := cqlSession.Query(q).Exec(); err != nil {
				return 0, db.WrapDbErrorWithQuery("cannot create idx table", q, err)
			}
			tablesCreated++
		}
	}

	logger.Info("created %d tables, creating messages to send for run %d...", tablesCreated, runId)

	for i := 0; i < len(intervals); i++ {
		now := time.Now()
		logger.Info("BatchStarted: [%d,%d]...", intervals[i][0], intervals[i][1])
		msg := wfmodel.Message{
			Id:              uuid.NewString(),
			Ts:              now.UnixMilli(),
			ScriptURL:       scriptFilePath,
			ScriptParamsURL: paramsFilePath,
			DataKeyspace:    keyspace,
			RunId:           runId,
			TargetNodeName:  nodeName,
			FirstToken:      intervals[i][0],
			LastToken:       intervals[i][1],
			BatchIdx:        int16(i),
			BatchesTotal:    int16(len(intervals))}

		if acknowledgerCmd := wf.ProcessDataBatchMsgNew(envConfig, logger, &msg, 0, nil); acknowledgerCmd != mq.AcknowledgerCmdAck {
			return 0, fmt.Errorf("processor returned acknowledgerCmd %d, assuming failure, check the logs", acknowledgerCmd)
		}
		logger.Info("BatchComplete: [%d,%d], %.3fs", intervals[i][0], intervals[i][1], time.Since(now).Seconds())
	}
	if err := wfdb.SetRunStatus(logger, cqlSession, keyspace, runId, wfmodel.RunComplete, fmt.Sprintf("Toolbelt RunNode(%s), run successful", nodeName), cql.IgnoreIfExists); err != nil {
		return 0, err
	}

	return runId, nil
}
