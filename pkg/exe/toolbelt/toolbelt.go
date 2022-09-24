package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/capillariesio/capillaries/pkg/api"
	"github.com/capillariesio/capillaries/pkg/cql"
	"github.com/capillariesio/capillaries/pkg/custom"
	"github.com/capillariesio/capillaries/pkg/env"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/sc"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
	amqp "github.com/rabbitmq/amqp091-go"
)

type DotDiagramType string

const (
	DotDiagramIndexes   DotDiagramType = "indexes"
	DotDiagramFields    DotDiagramType = "fields"
	DotDiagramRunStatus DotDiagramType = "run_status"
)

func NodeBatchStatusToColor(status wfmodel.NodeBatchStatusType) string {
	switch status {
	case wfmodel.NodeBatchNone:
		return "white"
	case wfmodel.NodeBatchStart:
		return "lightblue"
	case wfmodel.NodeBatchSuccess:
		return "green"
	case wfmodel.NodeBatchFail:
		return "red"
	case wfmodel.NodeBatchRunStopReceived:
		return "orangered"
	default:
		return "cyan"
	}
}

func GetDotDiagram(scriptDef *sc.ScriptDef, dotDiagramType DotDiagramType, nodeColorMap map[string]string) string {
	var b strings.Builder

	const RecordFontSize int = 20
	const ArrowFontSize int = 18

	urlEscaper := strings.NewReplacer(`{`, `\{`, `}`, `\}`, `|`, `\|`)
	b.WriteString(fmt.Sprintf("\ndigraph %s {\nrankdir=\"TD\";\n node [fontname=\"Helvetica\"];\nedge [fontname=\"Helvetica\"];\ngraph [splines=true, pad=\"0.5\", ranksep=\"0.5\", nodesep=\"0.5\"];\n", dotDiagramType))
	for _, node := range scriptDef.ScriptNodes {
		penWidth := "1"
		if node.StartPolicy == sc.NodeStartManual {
			penWidth = "6"
		}
		fillColor := "white"
		var ok bool
		if nodeColorMap != nil {
			if fillColor, ok = nodeColorMap[node.Name]; !ok {
				fillColor = "white" // This run does not affect this node, or the node was not started
			}
		}

		if node.HasFileReader() {
			arrowLabelBuilder := strings.Builder{}
			if dotDiagramType == DotDiagramType(DotDiagramFields) {
				for colName := range node.FileReader.Columns {
					arrowLabelBuilder.WriteString(colName)
					arrowLabelBuilder.WriteString("\\l")
				}
			}
			fileNames := make([]string, len(node.FileReader.SrcFileUrls))
			copy(fileNames, node.FileReader.SrcFileUrls)

			b.WriteString(fmt.Sprintf("\"%s\" -> \"%s\" [style=dotted, fontsize=\"%d\", label=\"%s\"];\n", node.FileReader.SrcFileUrls[0], node.GetTargetName(), ArrowFontSize, arrowLabelBuilder.String()))
			b.WriteString(fmt.Sprintf("\"%s\" [shape=folder, fontsize=\"%d\", label=\"%s\", tooltip=\"Source data file(s)\"];\n", node.FileReader.SrcFileUrls[0], RecordFontSize, strings.Join(fileNames, "\\n")))
		}

		allUsedFields := sc.FieldRefs{}

		if node.HasFileCreator() {
			usedInAllTargetFileExpressions := node.FileCreator.GetFieldRefsUsedInAllTargetFileExpressions()
			allUsedFields.Append(usedInAllTargetFileExpressions)
		} else if node.HasTableCreator() {
			usedInAllTargetTableExpressions := sc.GetFieldRefsUsedInAllTargetExpressions(node.TableCreator.Fields)
			allUsedFields.Append(usedInAllTargetTableExpressions)
		}

		if node.HasTableReader() {
			var inSrcArrowLabel string
			if dotDiagramType == DotDiagramType(DotDiagramIndexes) || dotDiagramType == DotDiagramType(DotDiagramRunStatus) {
				if node.TableReader.ExpectedBatchesTotal > 1 {
					inSrcArrowLabel = fmt.Sprintf("%s (%d batches)", node.TableReader.TableName, node.TableReader.ExpectedBatchesTotal)
				} else {
					inSrcArrowLabel = fmt.Sprintf("%s (no parallelism)", node.TableReader.TableName)
				}
			} else if dotDiagramType == DotDiagramType(DotDiagramFields) {
				inSrcArrowLabelBuilder := strings.Builder{}
				for i := 0; i < len(allUsedFields); i++ {
					if allUsedFields[i].TableName == sc.ReaderAlias {
						inSrcArrowLabelBuilder.WriteString(allUsedFields[i].FieldName)
						inSrcArrowLabelBuilder.WriteString("\\l")
					}
				}
				inSrcArrowLabel = inSrcArrowLabelBuilder.String()
			}
			if node.HasFileCreator() {
				// In (reader)
				//if node.TableReader.ExpectedBatchesTotal > 1 {
				//b.WriteString(fmt.Sprintf("\"%s\" -> \"%s\" [style=solid];\n", node.TableReader.TableName, node.Name))
				//b.WriteString(fmt.Sprintf("\"%s\" -> \"%s\" [style=solid];\n", node.TableReader.TableName, node.Name))
				//}
				b.WriteString(fmt.Sprintf("\"%s\" -> \"%s\" [style=solid, fontsize=\"%d\", label=\"%s\"];\n", node.TableReader.TableName, node.Name, ArrowFontSize, inSrcArrowLabel))

				// Node (file)
				b.WriteString(fmt.Sprintf("\"%s\" [shape=record, penwidth=\"%s\", fontsize=\"%d\", fillcolor=\"%s\", style=\"filled\", label=\"{%s|creates file:\\n%s}\", tooltip=\"%s\"];\n", node.Name, penWidth, RecordFontSize, fillColor, node.Name, urlEscaper.Replace(node.FileCreator.UrlTemplate), node.Desc))

				// Out (file)
				arrowLabelBuilder := strings.Builder{}
				if dotDiagramType == DotDiagramType(DotDiagramFields) {
					for i := 0; i < len(allUsedFields); i++ {
						arrowLabelBuilder.WriteString(allUsedFields[i].FieldName)
						arrowLabelBuilder.WriteString("\\l")
					}
				}

				b.WriteString(fmt.Sprintf("\"%s\" -> \"%s\" [style=dotted, fontsize=\"%d\", label=\"%s\"];\n", node.Name, node.FileCreator.UrlTemplate, ArrowFontSize, arrowLabelBuilder.String()))
				b.WriteString(fmt.Sprintf("\"%s\" [shape=note, fontsize=\"%d\", label=\"%s\", tooltip=\"Target data file(s)\"];\n", node.FileCreator.UrlTemplate, RecordFontSize, node.FileCreator.UrlTemplate))
			} else {
				// In (reader)
				// if node.TableReader.ExpectedBatchesTotal > 1 {
				// 	b.WriteString(fmt.Sprintf("\"%s\" -> \"%s\" [style=solid];\n", node.TableReader.TableName, node.GetTargetName()))
				// 	b.WriteString(fmt.Sprintf("\"%s\" -> \"%s\" [style=solid];\n", node.TableReader.TableName, node.GetTargetName()))
				// }
				b.WriteString(fmt.Sprintf("\"%s\" -> \"%s\" [style=solid, fontsize=\"%d\", label=\"%s\"];\n", node.TableReader.TableName, node.GetTargetName(), ArrowFontSize, inSrcArrowLabel))
			}

			if node.HasLookup() {
				inLkpArrowLabel := fmt.Sprintf("%s (lookup)", node.Lookup.IndexName)
				if dotDiagramType == DotDiagramType(DotDiagramFields) {
					inLkpArrowLabelBuilder := strings.Builder{}
					for i := 0; i < len(allUsedFields); i++ {
						if allUsedFields[i].TableName == sc.LookupAlias {
							inLkpArrowLabelBuilder.WriteString(allUsedFields[i].FieldName)
							inLkpArrowLabelBuilder.WriteString("\\l")
						}
					}
					inLkpArrowLabel = inLkpArrowLabelBuilder.String()
				}
				// In (lookup)
				b.WriteString(fmt.Sprintf("\"%s\" -> \"%s\" [style=dashed, fontsize=\"%d\", label=\"%s\"];\n", node.Lookup.TableCreator.Name, node.GetTargetName(), ArrowFontSize, inLkpArrowLabel))
			}

		}

		if node.HasTableCreator() {
			// Node (table)
			if node.HasLookup() {
				b.WriteString(fmt.Sprintf("\"%s\" [shape=record, penwidth=\"%s\", fontsize=\"%d\", fillcolor=\"%s\", style=\"filled\", label=\"{%s|creates table:\\n%s|group:%t, join:%s}\", tooltip=\"%s\"];\n", node.TableCreator.Name, penWidth, RecordFontSize, fillColor, node.Name, node.TableCreator.Name, node.Lookup.IsGroup, node.Lookup.LookupJoin, node.Desc))
			} else {
				b.WriteString(fmt.Sprintf("\"%s\" [shape=record, penwidth=\"%s\", fontsize=\"%d\", fillcolor=\"%s\", style=\"filled\", label=\"{%s|creates table:\\n%s}\", tooltip=\"%s\"];\n", node.TableCreator.Name, penWidth, RecordFontSize, fillColor, node.Name, node.TableCreator.Name, node.Desc))
			}
		}
	}
	b.WriteString("}\n")

	return b.String()
}

const LogTsFormatUnquoted = `2006-01-02T15:04:05.000-0700`

type StandardToolbeltProcessorDefFactory struct {
}

func (f *StandardToolbeltProcessorDefFactory) Create(processorType string) (sc.CustomProcessorDef, bool) {
	// All processors to be supported by this 'stock' binary (daemon/toolbelt).
	// If you develop your own processor(s), use your own ProcessorDefFactory that lists all processors,
	// they all must implement CustomProcessorRunner interface
	switch processorType {
	case custom.ProcessorPyCalcName:
		return &custom.PyCalcProcessorDef{}, true
	case custom.ProcessorTagAndDenormalizeName:
		return &custom.TagAndDenormalizeProcessorDef{}, true
	default:
		return nil, false
	}
}

func stringToArrayOfInt16(s string) ([]int16, error) {
	var result []int16
	if len(strings.TrimSpace(s)) > 0 {
		stringItems := strings.Split(s, ",")
		result = make([]int16, len(stringItems))
		for itemIdx, stringItem := range stringItems {
			intItem, err := strconv.ParseInt(strings.TrimSpace(stringItem), 10, 16)
			if err != nil {
				return nil, fmt.Errorf("invalid int16 %s:%s", stringItem, err.Error())
			}
			result[itemIdx] = int16(intItem)
		}
	}
	return result, nil
}

func stringToArrayOfStrings(s string) ([]string, error) {
	var result []string
	if len(strings.TrimSpace(s)) > 0 {
		stringItems := strings.Split(s, ",")
		result = make([]string, len(stringItems))
		for itemIdx, stringItem := range stringItems {
			result[itemIdx] = stringItem
		}
	}
	return result, nil
}

func main() {
	//defer profile.Start().Stop()

	envConfig, err := env.ReadEnvConfigFile("env_config.json")
	if err != nil {
		log.Fatalf(err.Error())
		os.Exit(1)
	}
	envConfig.CustomProcessorDefFactoryInstance = &StandardToolbeltProcessorDefFactory{}
	logger, err := l.NewLoggerFromEnvConfig(envConfig)
	if err != nil {
		log.Fatalf(err.Error())
		os.Exit(1)
	}
	defer logger.Close()

	if len(os.Args) <= 1 {
		log.Fatalf("COMMAND EXPECTED")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "validate_script":
		validateScriptCmd := flag.NewFlagSet("validate_script", flag.ExitOnError)
		scriptFilePath := validateScriptCmd.String("script_file", "", "Path to script file")
		paramsFilePath := validateScriptCmd.String("params_file", "", "Path to script parameters map file")
		isIdxDag := validateScriptCmd.Bool("idx_dag", false, "Print index DAG")
		isFieldDag := validateScriptCmd.Bool("field_dag", false, "Print field DAG")
		validateScriptCmd.Parse(os.Args[2:])

		script, err := sc.NewScriptFromFiles(envConfig.CaPath, *scriptFilePath, *paramsFilePath, envConfig.CustomProcessorDefFactoryInstance, envConfig.CustomProcessorsSettings)
		if err != nil {
			log.Fatalf(err.Error())
			os.Exit(1)
		}

		if *isIdxDag {
			fmt.Println(GetDotDiagram(script, DotDiagramIndexes, nil))
		}
		if *isFieldDag {
			fmt.Println(GetDotDiagram(script, DotDiagramFields, nil))
		}

	case "stop_run":
		stopRunCmd := flag.NewFlagSet("stop_run", flag.ExitOnError)
		keyspace := stopRunCmd.String("keyspace", "", "Keyspace (session id)")
		runIdString := stopRunCmd.String("run_id", "", "Run id")
		stopRunCmd.Parse(os.Args[2:])

		runId, err := strconv.ParseInt(strings.TrimSpace(*runIdString), 10, 16)
		if err != nil {
			log.Fatalf(err.Error())
			os.Exit(1)
		}

		cqlSession, err := cql.NewSession(envConfig, *keyspace)
		if err != nil {
			log.Fatalf(err.Error())
			os.Exit(1)
		}

		err = api.StopRun(logger, cqlSession, *keyspace, int16(runId), "stopped by toolbelt")
		if err != nil {
			log.Fatalf(err.Error())
			os.Exit(1)
		}

	case "get_run_history":
		getRunsCmd := flag.NewFlagSet("get_runs", flag.ExitOnError)
		keyspace := getRunsCmd.String("keyspace", "", "Keyspace (session id)")
		getRunsCmd.Parse(os.Args[2:])

		cqlSession, err := cql.NewSession(envConfig, *keyspace)
		if err != nil {
			log.Fatalf(err.Error())
			os.Exit(1)
		}

		runs, err := api.GetRunHistory(logger, cqlSession, *keyspace)
		if err != nil {
			log.Fatalf(err.Error())
			os.Exit(1)
		}
		fmt.Println(strings.Join(wfmodel.RunHistoryAllFields(), ","))
		for _, r := range runs {
			fmt.Printf("%s,%d,%d,%s\n", r.Ts.Format(LogTsFormatUnquoted), r.RunId, r.Status, strings.ReplaceAll(r.Comment, ",", ";"))
		}

	case "get_node_history":
		getNodeHistoryCmd := flag.NewFlagSet("get_node_history", flag.ExitOnError)
		keyspace := getNodeHistoryCmd.String("keyspace", "", "Keyspace (session id)")
		runIdsString := getNodeHistoryCmd.String("run_ids", "", "Limit results to specific run ids (optional), comma-separated list")
		getNodeHistoryCmd.Parse(os.Args[2:])

		cqlSession, err := cql.NewSession(envConfig, *keyspace)
		if err != nil {
			log.Fatalf(err.Error())
			os.Exit(1)
		}

		runIds, err := stringToArrayOfInt16(*runIdsString)
		if err != nil {
			log.Fatalf(err.Error())
			os.Exit(1)
		}

		nodes, err := api.GetNodeHistory(logger, cqlSession, *keyspace, runIds)
		if err != nil {
			log.Fatalf(err.Error())
			os.Exit(1)
		}
		fmt.Println(strings.Join(wfmodel.NodeHistoryAllFields(), ","))
		for _, n := range nodes {
			fmt.Printf("%s,%d,%s,%d,%s\n", n.Ts.Format(LogTsFormatUnquoted), n.RunId, n.ScriptNode, n.Status, strings.ReplaceAll(n.Comment, ",", ";"))
		}

	case "get_batch_history":
		getBatchHistoryCmd := flag.NewFlagSet("get_batch_history", flag.ExitOnError)
		keyspace := getBatchHistoryCmd.String("keyspace", "", "Keyspace (session id)")
		runIdsString := getBatchHistoryCmd.String("run_ids", "", "Limit results to specific run ids (optional), comma-separated list")
		nodeNamesString := getBatchHistoryCmd.String("nodes", "", "Limit results to specific node names (optional), comma-separated list")
		getBatchHistoryCmd.Parse(os.Args[2:])

		cqlSession, err := cql.NewSession(envConfig, *keyspace)
		if err != nil {
			log.Fatalf(err.Error())
			os.Exit(1)
		}

		runIds, err := stringToArrayOfInt16(*runIdsString)
		if err != nil {
			log.Fatalf(err.Error())
			os.Exit(1)
		}

		nodeNames, err := stringToArrayOfStrings(*nodeNamesString)
		if err != nil {
			log.Fatalf(err.Error())
			os.Exit(1)
		}

		runs, err := api.GetBatchHistory(logger, cqlSession, *keyspace, runIds, nodeNames)
		if err != nil {
			log.Fatalf(err.Error())
			os.Exit(1)
		}

		fmt.Println(strings.Join(wfmodel.BatchHistoryAllFields(), ","))
		for _, r := range runs {
			fmt.Printf("%s,%d,%s,%d,%d,%d,%d,%d,%s\n", r.Ts.Format(LogTsFormatUnquoted), r.RunId, r.ScriptNode, r.BatchIdx, r.BatchesTotal, r.Status, r.FirstToken, r.LastToken, strings.ReplaceAll(r.Comment, ",", ";"))
		}

	case "get_table_cql":
		getTableCqlCmd := flag.NewFlagSet("get_table_cql", flag.ExitOnError)
		scriptFilePath := getTableCqlCmd.String("script_file", "", "Path to script file")
		paramsFilePath := getTableCqlCmd.String("params_file", "", "Path to script parameters map file")
		keyspace := getTableCqlCmd.String("keyspace", "", "Keyspace (session id)")
		runId := getTableCqlCmd.Int("run_id", 0, "Run id")
		startNodesString := getTableCqlCmd.String("start_nodes", "", "Comma-separated list of start node names")
		getTableCqlCmd.Parse(os.Args[2:])

		script, err := sc.NewScriptFromFiles(envConfig.CaPath, *scriptFilePath, *paramsFilePath, envConfig.CustomProcessorDefFactoryInstance, envConfig.CustomProcessorsSettings)
		if err != nil {
			log.Fatalf(err.Error())
			os.Exit(1)
		}

		startNodes := strings.Split(*startNodesString, ",")

		fmt.Print(api.GetTablesCql(script, *keyspace, int16(*runId), startNodes))

	case "get_run_status_diagram":
		getRunStatusDiagramCmd := flag.NewFlagSet("get_run_status_diagram", flag.ExitOnError)
		scriptFilePath := getRunStatusDiagramCmd.String("script_file", "", "Path to script file")
		paramsFilePath := getRunStatusDiagramCmd.String("params_file", "", "Path to script parameters map file")
		keyspace := getRunStatusDiagramCmd.String("keyspace", "", "Keyspace (session id)")
		runIdString := getRunStatusDiagramCmd.String("run_id", "", "Run id")
		getRunStatusDiagramCmd.Parse(os.Args[2:])

		runId, err := strconv.ParseInt(strings.TrimSpace(*runIdString), 10, 16)
		if err != nil {
			log.Fatalf(err.Error())
			os.Exit(1)
		}

		script, err := sc.NewScriptFromFiles(envConfig.CaPath, *scriptFilePath, *paramsFilePath, envConfig.CustomProcessorDefFactoryInstance, envConfig.CustomProcessorsSettings)
		if err != nil {
			log.Fatalf(err.Error())
			os.Exit(1)
		}

		cqlSession, err := cql.NewSession(envConfig, *keyspace)
		if err != nil {
			log.Fatalf(err.Error())
			os.Exit(1)
		}

		nodes, err := api.GetNodeHistory(logger, cqlSession, *keyspace, []int16{int16(runId)})
		if err != nil {
			log.Fatalf(err.Error())
			os.Exit(1)
		}

		nodeColorMap := map[string]string{}
		for _, node := range nodes {
			nodeColorMap[node.ScriptNode] = NodeBatchStatusToColor(node.Status)
		}

		fmt.Println(GetDotDiagram(script, DotDiagramRunStatus, nodeColorMap))

	case "drop_keyspace":
		dropKsCmd := flag.NewFlagSet("drop_keyspae", flag.ExitOnError)
		keyspace := dropKsCmd.String("keyspace", "", "Keyspace (session id)")
		dropKsCmd.Parse(os.Args[2:])

		cqlSession, err := cql.NewSession(envConfig, *keyspace)
		if err != nil {
			log.Fatalf(err.Error())
			os.Exit(1)
		}

		err = api.DropKeyspace(logger, cqlSession, *keyspace)
		if err != nil {
			log.Fatalf(err.Error())
			os.Exit(1)
		}

	case "start_run":
		startRunCmd := flag.NewFlagSet("start_run", flag.ExitOnError)
		keyspace := startRunCmd.String("keyspace", "", "Keyspace (session id)")
		scriptFilePath := startRunCmd.String("script_file", "", "Path to script file")
		paramsFilePath := startRunCmd.String("params_file", "", "Path to script parameters map file")
		startNodesString := startRunCmd.String("start_nodes", "", "Comma-separated list of start node names")
		startRunCmd.Parse(os.Args[2:])

		startNodes := strings.Split(*startNodesString, ",")

		cqlSession, err := cql.NewSession(envConfig, *keyspace)
		if err != nil {
			log.Fatalf(err.Error())
			os.Exit(1)
		}

		// RabbitMQ boilerplate
		amqpConnection, err := amqp.Dial(envConfig.Amqp.URL)
		if err != nil {
			log.Fatalf(fmt.Sprintf("cannot dial RabbitMQ at %v, will reconnect: %v\n", envConfig.Amqp.URL, err))
			os.Exit(1)
		}
		defer amqpConnection.Close()

		amqpChannel, err := amqpConnection.Channel()
		if err != nil {
			log.Fatalf(fmt.Sprintf("cannot create amqp channel, will reconnect: %v\n", err))
			os.Exit(1)
		}
		defer amqpChannel.Close()

		runId, err := api.StartRun(envConfig, logger, amqpChannel, *scriptFilePath, *paramsFilePath, cqlSession, *keyspace, startNodes)
		if err != nil {
			log.Fatalf(err.Error())
			os.Exit(1)
		}

		fmt.Println(runId)

	case "exec_node":
		execNodeCmd := flag.NewFlagSet("exec_node", flag.ExitOnError)
		keyspace := execNodeCmd.String("keyspace", "", "Keyspace (session id)")
		scriptFilePath := execNodeCmd.String("script_file", "", "Path to script file")
		paramsFilePath := execNodeCmd.String("params_file", "", "Path to script parameters map file")
		runIdParam := execNodeCmd.Int("run_id", 0, "run id (optional, use with extra caution as it will modify existing run id results)")
		nodeName := execNodeCmd.String("node_id", "", "Script node name")
		execNodeCmd.Parse(os.Args[2:])

		runId := int16(*runIdParam)

		startTime := time.Now()

		cqlSession, err := cql.NewSession(envConfig, *keyspace)
		if err != nil {
			log.Fatalf(err.Error())
			os.Exit(1)
		}

		runId, err = api.RunNode(envConfig, logger, *nodeName, runId, *scriptFilePath, *paramsFilePath, cqlSession, *keyspace)
		if err != nil {
			log.Fatalf(err.Error())
			os.Exit(1)
		}
		fmt.Printf("run %d, elapsed %v\n", runId, time.Since(startTime))

	default:
		fmt.Println("expected some valid command")
		os.Exit(1)
	}
}
