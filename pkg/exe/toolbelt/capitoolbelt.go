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
	"github.com/capillariesio/capillaries/pkg/custom/py_calc"
	"github.com/capillariesio/capillaries/pkg/custom/tag_and_denormalize"
	"github.com/capillariesio/capillaries/pkg/db"
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

func GetDotDiagram(scriptDef *sc.ScriptDef, dotDiagramType DotDiagramType, nodeColorMap map[string]string) string { //nolint:all cognitive complexity 82
	var b strings.Builder

	const recordFontSize int = 20
	const arrowFontSize int = 18

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

			b.WriteString(fmt.Sprintf("\"%s\" -> \"%s\" [style=dotted, fontsize=\"%d\", label=\"%s\"];\n", node.FileReader.SrcFileUrls[0], node.GetTargetName(), arrowFontSize, arrowLabelBuilder.String()))
			b.WriteString(fmt.Sprintf("\"%s\" [shape=folder, fontsize=\"%d\", label=\"%s\", tooltip=\"Source data file(s)\"];\n", node.FileReader.SrcFileUrls[0], recordFontSize, strings.Join(fileNames, "\\n")))
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
				b.WriteString(fmt.Sprintf("\"%s\" -> \"%s\" [style=solid, fontsize=\"%d\", label=\"%s\"];\n", node.TableReader.TableName, node.Name, arrowFontSize, inSrcArrowLabel))

				// Node (file)
				b.WriteString(fmt.Sprintf("\"%s\" [shape=record, penwidth=\"%s\", fontsize=\"%d\", fillcolor=\"%s\", style=\"filled\", label=\"{%s|creates file:\\n%s}\", tooltip=\"%s\"];\n", node.Name, penWidth, recordFontSize, fillColor, node.Name, urlEscaper.Replace(node.FileCreator.UrlTemplate), node.Desc))

				// Out (file)
				arrowLabelBuilder := strings.Builder{}
				if dotDiagramType == DotDiagramType(DotDiagramFields) {
					for i := 0; i < len(allUsedFields); i++ {
						arrowLabelBuilder.WriteString(allUsedFields[i].FieldName)
						arrowLabelBuilder.WriteString("\\l")
					}
				}

				b.WriteString(fmt.Sprintf("\"%s\" -> \"%s\" [style=dotted, fontsize=\"%d\", label=\"%s\"];\n", node.Name, node.FileCreator.UrlTemplate, arrowFontSize, arrowLabelBuilder.String()))
				b.WriteString(fmt.Sprintf("\"%s\" [shape=note, fontsize=\"%d\", label=\"%s\", tooltip=\"Target data file(s)\"];\n", node.FileCreator.UrlTemplate, recordFontSize, node.FileCreator.UrlTemplate))
			} else {
				b.WriteString(fmt.Sprintf("\"%s\" -> \"%s\" [style=solid, fontsize=\"%d\", label=\"%s\"];\n", node.TableReader.TableName, node.GetTargetName(), arrowFontSize, inSrcArrowLabel))
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
				b.WriteString(fmt.Sprintf("\"%s\" -> \"%s\" [style=dashed, fontsize=\"%d\", label=\"%s\"];\n", node.Lookup.TableCreator.Name, node.GetTargetName(), arrowFontSize, inLkpArrowLabel))
			}

		}

		if node.HasTableCreator() {
			// Node (table)
			if node.HasLookup() {
				b.WriteString(fmt.Sprintf("\"%s\" [shape=record, penwidth=\"%s\", fontsize=\"%d\", fillcolor=\"%s\", style=\"filled\", label=\"{%s|creates table:\\n%s|group:%t, join:%s}\", tooltip=\"%s\"];\n", node.TableCreator.Name, penWidth, recordFontSize, fillColor, node.Name, node.TableCreator.Name, node.Lookup.IsGroup, node.Lookup.LookupJoin, node.Desc))
			} else {
				b.WriteString(fmt.Sprintf("\"%s\" [shape=record, penwidth=\"%s\", fontsize=\"%d\", fillcolor=\"%s\", style=\"filled\", label=\"{%s|creates table:\\n%s}\", tooltip=\"%s\"];\n", node.TableCreator.Name, penWidth, recordFontSize, fillColor, node.Name, node.TableCreator.Name, node.Desc))
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
	case py_calc.ProcessorPyCalcName:
		return &py_calc.PyCalcProcessorDef{}, true
	case tag_and_denormalize.ProcessorTagAndDenormalizeName:
		return &tag_and_denormalize.TagAndDenormalizeProcessorDef{}, true
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

const (
	CmdValidateScript      string = "validate_script"
	CmdStartRun            string = "start_run"
	CmdStopRun             string = "stop_run"
	CmdExecNode            string = "exec_node"
	CmdGetRunHistory       string = "get_run_history"
	CmdGetNodeHistory      string = "get_node_history"
	CmdGetBatchHistory     string = "get_batch_history"
	CmdGetRunStatusDiagram string = "drop_run_status_diagram"
	CmdDropKeyspace        string = "drop_keyspace"
	CmdGetTableCql         string = "get_table_cql"
)

func usage(flagset *flag.FlagSet) {
	fmt.Printf("Capillaries toolbelt\nUsage: capitoolbelt <command> <command parameters>\nCommands:\n")
	fmt.Printf("  %s\n  %s\n  %s\n  %s\n  %s\n  %s\n  %s\n  %s\n  %s\n  %s\n",
		CmdValidateScript,
		CmdStartRun,
		CmdStopRun,
		CmdExecNode,
		CmdGetRunHistory,
		CmdGetNodeHistory,
		CmdGetBatchHistory,
		CmdGetRunStatusDiagram,
		CmdDropKeyspace,
		CmdGetTableCql)
	if flagset != nil {
		fmt.Printf("\n%s parameters:\n", flagset.Name())
		flagset.PrintDefaults()
	}
}

func main() { //nolint:all cognitive complexity 90
	// defer profile.Start().Stop()

	envConfig, err := env.ReadEnvConfigFile("capitoolbelt.json")
	if err != nil {
		log.Fatalf(err.Error())
	}
	envConfig.CustomProcessorDefFactoryInstance = &StandardToolbeltProcessorDefFactory{}
	logger, err := l.NewLoggerFromEnvConfig(envConfig)
	if err != nil {
		log.Fatalf(err.Error())
	}
	defer logger.Close()

	if len(os.Args) <= 1 {
		usage(nil)
		os.Exit(0)
	}

	switch os.Args[1] {
	case CmdValidateScript:
		validateScriptCmd := flag.NewFlagSet(CmdValidateScript, flag.ExitOnError)
		scriptFilePath := validateScriptCmd.String("script_file", "", "Path to script file")
		paramsFilePath := validateScriptCmd.String("params_file", "", "Path to script parameters map file")
		isIdxDag := validateScriptCmd.Bool("idx_dag", false, "Print index DAG")
		isFieldDag := validateScriptCmd.Bool("field_dag", false, "Print field DAG")
		if err := validateScriptCmd.Parse(os.Args[2:]); err != nil || *scriptFilePath == "" || *paramsFilePath == "" || (!*isIdxDag && !*isFieldDag) {
			usage(validateScriptCmd)
			os.Exit(0)
		}

		script, _, err := sc.NewScriptFromFiles(envConfig.CaPath, envConfig.PrivateKeys, *scriptFilePath, *paramsFilePath, envConfig.CustomProcessorDefFactoryInstance, envConfig.CustomProcessorsSettings)
		if err != nil {
			log.Fatalf(err.Error())
		}

		if *isIdxDag {
			fmt.Println(GetDotDiagram(script, DotDiagramIndexes, nil))
		}
		if *isFieldDag {
			fmt.Println(GetDotDiagram(script, DotDiagramFields, nil))
		}

	case CmdStopRun:
		stopRunCmd := flag.NewFlagSet(CmdStopRun, flag.ExitOnError)
		keyspace := stopRunCmd.String("keyspace", "", "Keyspace (session id)")
		runIdString := stopRunCmd.String("run_id", "", "Run id")
		if err := stopRunCmd.Parse(os.Args[2:]); err != nil {
			usage(stopRunCmd)
			os.Exit(0)
		}

		runId, err := strconv.ParseInt(strings.TrimSpace(*runIdString), 10, 16)
		if err != nil {
			log.Fatalf(err.Error())
		}

		cqlSession, err := db.NewSession(envConfig, *keyspace, db.DoNotCreateKeyspaceOnConnect)
		if err != nil {
			log.Fatalf(err.Error())
		}

		err = api.StopRun(logger, cqlSession, *keyspace, int16(runId), "stopped by toolbelt")
		if err != nil {
			log.Fatalf(err.Error())
		}

	case CmdGetRunHistory:
		getRunsCmd := flag.NewFlagSet(CmdGetRunHistory, flag.ExitOnError)
		keyspace := getRunsCmd.String("keyspace", "", "Keyspace (session id)")
		if err := getRunsCmd.Parse(os.Args[2:]); err != nil {
			usage(getRunsCmd)
			os.Exit(0)
		}

		cqlSession, err := db.NewSession(envConfig, *keyspace, db.DoNotCreateKeyspaceOnConnect)
		if err != nil {
			log.Fatalf(err.Error())
		}

		runs, err := api.GetRunHistory(logger, cqlSession, *keyspace)
		if err != nil {
			log.Fatalf(err.Error())
		}
		fmt.Println(strings.Join(wfmodel.RunHistoryEventAllFields(), ","))
		for _, r := range runs {
			fmt.Printf("%s,%d,%d,%s\n", r.Ts.Format(LogTsFormatUnquoted), r.RunId, r.Status, strings.ReplaceAll(r.Comment, ",", ";"))
		}

	case CmdGetNodeHistory:
		getNodeHistoryCmd := flag.NewFlagSet(CmdGetNodeHistory, flag.ExitOnError)
		keyspace := getNodeHistoryCmd.String("keyspace", "", "Keyspace (session id)")
		runIdsString := getNodeHistoryCmd.String("run_ids", "", "Limit results to specific run ids (optional), comma-separated list")
		if err := getNodeHistoryCmd.Parse(os.Args[2:]); err != nil {
			usage(getNodeHistoryCmd)
			os.Exit(0)
		}

		cqlSession, err := db.NewSession(envConfig, *keyspace, db.DoNotCreateKeyspaceOnConnect)
		if err != nil {
			log.Fatalf(err.Error())
		}

		runIds, err := stringToArrayOfInt16(*runIdsString)
		if err != nil {
			log.Fatalf(err.Error())
		}

		nodes, err := api.GetRunsNodeHistory(logger, cqlSession, *keyspace, runIds)
		if err != nil {
			log.Fatalf(err.Error())
		}
		fmt.Println(strings.Join(wfmodel.NodeHistoryEventAllFields(), ","))
		for _, n := range nodes {
			fmt.Printf("%s,%d,%s,%d,%s\n", n.Ts.Format(LogTsFormatUnquoted), n.RunId, n.ScriptNode, n.Status, strings.ReplaceAll(n.Comment, ",", ";"))
		}

	case CmdGetBatchHistory:
		getBatchHistoryCmd := flag.NewFlagSet(CmdGetBatchHistory, flag.ExitOnError)
		keyspace := getBatchHistoryCmd.String("keyspace", "", "Keyspace (session id)")
		runIdsString := getBatchHistoryCmd.String("run_ids", "", "Limit results to specific run ids (optional), comma-separated list")
		nodeNamesString := getBatchHistoryCmd.String("nodes", "", "Limit results to specific node names (optional), comma-separated list")
		if err := getBatchHistoryCmd.Parse(os.Args[2:]); err != nil {
			usage(getBatchHistoryCmd)
			os.Exit(0)
		}

		cqlSession, err := db.NewSession(envConfig, *keyspace, db.DoNotCreateKeyspaceOnConnect)
		if err != nil {
			log.Fatalf(err.Error())
		}

		runIds, err := stringToArrayOfInt16(*runIdsString)
		if err != nil {
			log.Fatalf(err.Error())
		}

		var nodeNames []string
		if len(strings.TrimSpace(*nodeNamesString)) > 0 {
			nodeNames = strings.Split(*nodeNamesString, ",")
		} else {
			nodeNames = make([]string, 0)
		}

		runs, err := api.GetBatchHistory(logger, cqlSession, *keyspace, runIds, nodeNames)
		if err != nil {
			log.Fatalf(err.Error())
		}

		fmt.Println(strings.Join(wfmodel.BatchHistoryEventAllFields(), ","))
		for _, r := range runs {
			fmt.Printf("%s,%d,%s,%d,%d,%d,%d,%d,%s\n", r.Ts.Format(LogTsFormatUnquoted), r.RunId, r.ScriptNode, r.BatchIdx, r.BatchesTotal, r.Status, r.FirstToken, r.LastToken, strings.ReplaceAll(r.Comment, ",", ";"))
		}

	case CmdGetTableCql:
		getTableCqlCmd := flag.NewFlagSet(CmdGetTableCql, flag.ExitOnError)
		scriptFilePath := getTableCqlCmd.String("script_file", "", "Path to script file")
		paramsFilePath := getTableCqlCmd.String("params_file", "", "Path to script parameters map file")
		keyspace := getTableCqlCmd.String("keyspace", "", "Keyspace (session id)")
		runId := getTableCqlCmd.Int("run_id", 0, "Run id")
		startNodesString := getTableCqlCmd.String("start_nodes", "", "Comma-separated list of start node names")
		if err := getTableCqlCmd.Parse(os.Args[2:]); err != nil {
			usage(getTableCqlCmd)
			os.Exit(0)
		}

		script, _, err := sc.NewScriptFromFiles(envConfig.CaPath, envConfig.PrivateKeys, *scriptFilePath, *paramsFilePath, envConfig.CustomProcessorDefFactoryInstance, envConfig.CustomProcessorsSettings)
		if err != nil {
			log.Fatalf(err.Error())
		}

		startNodes := strings.Split(*startNodesString, ",")

		fmt.Print(api.GetTablesCql(script, *keyspace, int16(*runId), startNodes))

	case CmdGetRunStatusDiagram:
		getRunStatusDiagramCmd := flag.NewFlagSet(CmdGetRunStatusDiagram, flag.ExitOnError)
		scriptFilePath := getRunStatusDiagramCmd.String("script_file", "", "Path to script file")
		paramsFilePath := getRunStatusDiagramCmd.String("params_file", "", "Path to script parameters map file")
		keyspace := getRunStatusDiagramCmd.String("keyspace", "", "Keyspace (session id)")
		runIdString := getRunStatusDiagramCmd.String("run_id", "", "Run id")
		if err := getRunStatusDiagramCmd.Parse(os.Args[2:]); err != nil {
			usage(getRunStatusDiagramCmd)
			os.Exit(0)
		}

		runId, err := strconv.ParseInt(strings.TrimSpace(*runIdString), 10, 16)
		if err != nil {
			log.Fatalf(err.Error())
		}

		script, _, err := sc.NewScriptFromFiles(envConfig.CaPath, envConfig.PrivateKeys, *scriptFilePath, *paramsFilePath, envConfig.CustomProcessorDefFactoryInstance, envConfig.CustomProcessorsSettings)
		if err != nil {
			log.Fatalf(err.Error())
		}

		cqlSession, err := db.NewSession(envConfig, *keyspace, db.DoNotCreateKeyspaceOnConnect)
		if err != nil {
			log.Fatalf(err.Error())
		}

		nodes, err := api.GetRunsNodeHistory(logger, cqlSession, *keyspace, []int16{int16(runId)})
		if err != nil {
			log.Fatalf(err.Error())
		}

		nodeColorMap := map[string]string{}
		for _, node := range nodes {
			nodeColorMap[node.ScriptNode] = NodeBatchStatusToColor(node.Status)
		}

		fmt.Println(GetDotDiagram(script, DotDiagramRunStatus, nodeColorMap))

	case CmdDropKeyspace:
		dropKsCmd := flag.NewFlagSet(CmdDropKeyspace, flag.ExitOnError)
		keyspace := dropKsCmd.String("keyspace", "", "Keyspace (session id)")
		if err := dropKsCmd.Parse(os.Args[2:]); err != nil {
			usage(dropKsCmd)
			os.Exit(0)
		}

		cqlSession, err := db.NewSession(envConfig, *keyspace, db.DoNotCreateKeyspaceOnConnect)
		if err != nil {
			log.Fatalf(err.Error())
		}

		err = api.DropKeyspace(logger, cqlSession, *keyspace)
		if err != nil {
			log.Fatalf(err.Error())
		}

	case CmdStartRun:
		startRunCmd := flag.NewFlagSet(CmdStartRun, flag.ExitOnError)
		keyspace := startRunCmd.String("keyspace", "", "Keyspace (session id)")
		scriptFilePath := startRunCmd.String("script_file", "", "Path to script file")
		paramsFilePath := startRunCmd.String("params_file", "", "Path to script parameters map file")
		startNodesString := startRunCmd.String("start_nodes", "", "Comma-separated list of start node names")
		if err := startRunCmd.Parse(os.Args[2:]); err != nil {
			usage(startRunCmd)
			os.Exit(0)
		}

		startNodes := strings.Split(*startNodesString, ",")

		cqlSession, err := db.NewSession(envConfig, *keyspace, db.CreateKeyspaceOnConnect)
		if err != nil {
			log.Fatalf(err.Error())
		}

		// RabbitMQ boilerplate
		amqpConnection, err := amqp.Dial(envConfig.Amqp.URL)
		if err != nil {
			log.Fatalf(fmt.Sprintf("cannot dial RabbitMQ at %v, will reconnect: %v\n", envConfig.Amqp.URL, err))
		}
		defer amqpConnection.Close()

		amqpChannel, err := amqpConnection.Channel()
		if err != nil {
			log.Fatalf(fmt.Sprintf("cannot create amqp channel, will reconnect: %v\n", err))
		}
		defer amqpChannel.Close()

		runId, err := api.StartRun(envConfig, logger, amqpChannel, *scriptFilePath, *paramsFilePath, cqlSession, *keyspace, startNodes, "started by Toolbelt")
		if err != nil {
			log.Fatalf(err.Error())
		}

		fmt.Println(runId)

	case CmdExecNode:
		execNodeCmd := flag.NewFlagSet(CmdExecNode, flag.ExitOnError)
		keyspace := execNodeCmd.String("keyspace", "", "Keyspace (session id)")
		scriptFilePath := execNodeCmd.String("script_file", "", "Path to script file")
		paramsFilePath := execNodeCmd.String("params_file", "", "Path to script parameters map file")
		runIdParam := execNodeCmd.Int("run_id", 0, "run id (optional, use with extra caution as it will modify existing run id results)")
		nodeName := execNodeCmd.String("node_id", "", "Script node name")
		if err := execNodeCmd.Parse(os.Args[2:]); err != nil {
			usage(execNodeCmd)
			os.Exit(0)
		}

		runId := int16(*runIdParam)

		startTime := time.Now()

		cqlSession, err := db.NewSession(envConfig, *keyspace, db.CreateKeyspaceOnConnect)
		if err != nil {
			log.Fatalf(err.Error())
		}

		runId, err = api.RunNode(envConfig, logger, *nodeName, runId, *scriptFilePath, *paramsFilePath, cqlSession, *keyspace)
		if err != nil {
			log.Fatalf(err.Error())
		}
		fmt.Printf("run %d, elapsed %v\n", runId, time.Since(startTime))

	default:
		fmt.Printf("invalid command: %s\n", os.Args[1])
		usage(nil)
		os.Exit(1)
	}
}
