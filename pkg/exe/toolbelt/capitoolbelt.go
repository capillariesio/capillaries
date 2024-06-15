package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/capillariesio/capillaries/pkg/api"
	"github.com/capillariesio/capillaries/pkg/custom/py_calc"
	"github.com/capillariesio/capillaries/pkg/custom/tag_and_denormalize"
	"github.com/capillariesio/capillaries/pkg/db"
	"github.com/capillariesio/capillaries/pkg/env"
	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/sc"
	"github.com/capillariesio/capillaries/pkg/storage"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
)

type DotDiagramType string

const (
	DotDiagramIndexes   DotDiagramType = "indexes"
	DotDiagramFields    DotDiagramType = "fields"
	DotDiagramRunStatus DotDiagramType = "run_status"
)

func nodeBatchStatusToColor(status wfmodel.NodeBatchStatusType) string {
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

func drawFileReader(node *sc.ScriptNodeDef, dotDiagramType DotDiagramType, arrowFontSize int, recordFontSize int) string {
	var b strings.Builder
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
	return b.String()
}

func drawFileCreator(node *sc.ScriptNodeDef, dotDiagramType DotDiagramType, arrowFontSize int, recordFontSize int, allUsedFields sc.FieldRefs, penWidth string, fillColor string, urlEscaper *strings.Replacer, inSrcArrowLabel string) string {
	var b strings.Builder
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
	return b.String()
}

func drawTableReader(node *sc.ScriptNodeDef, dotDiagramType DotDiagramType, arrowFontSize int, recordFontSize int, allUsedFields sc.FieldRefs, penWidth string, fillColor string, urlEscaper *strings.Replacer) string {
	var b strings.Builder
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
		b.WriteString(drawFileCreator(node, dotDiagramType, arrowFontSize, recordFontSize, allUsedFields, penWidth, fillColor, urlEscaper, inSrcArrowLabel))
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
	return b.String()
}

func drawTableCreator(node *sc.ScriptNodeDef, recordFontSize int, penWidth string, fillColor string) string {
	if node.HasLookup() {
		return fmt.Sprintf("\"%s\" [shape=record, penwidth=\"%s\", fontsize=\"%d\", fillcolor=\"%s\", style=\"filled\", label=\"{%s|creates table:\\n%s|group:%t, join:%s}\", tooltip=\"%s\"];\n",
			node.TableCreator.Name, penWidth, recordFontSize, fillColor, node.Name, node.TableCreator.Name, node.Lookup.IsGroup, node.Lookup.LookupJoin, node.Desc)
	}
	return fmt.Sprintf("\"%s\" [shape=record, penwidth=\"%s\", fontsize=\"%d\", fillcolor=\"%s\", style=\"filled\", label=\"{%s|creates table:\\n%s}\", tooltip=\"%s\"];\n",
		node.TableCreator.Name, penWidth, recordFontSize, fillColor, node.Name, node.TableCreator.Name, node.Desc)
}

func getDotDiagram(scriptDef *sc.ScriptDef, dotDiagramType DotDiagramType, nodeColorMap map[string]string) string {
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
			b.WriteString(drawFileReader(node, dotDiagramType, arrowFontSize, recordFontSize))
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
			b.WriteString(drawTableReader(node, dotDiagramType, arrowFontSize, recordFontSize, allUsedFields, penWidth, fillColor, urlEscaper))
		}

		if node.HasTableCreator() {
			b.WriteString(drawTableCreator(node, recordFontSize, penWidth, fillColor))
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
	CmdValidateScript         string = "validate_script"
	CmdStartRun               string = "start_run"
	CmdStopRun                string = "stop_run"
	CmdExecNode               string = "exec_node"
	CmdGetRunHistory          string = "get_run_history"
	CmdGetNodeHistory         string = "get_node_history"
	CmdGetBatchHistory        string = "get_batch_history"
	CmdGetRunStatusDiagram    string = "drop_run_status_diagram"
	CmdDropKeyspace           string = "drop_keyspace"
	CmdGetTableCql            string = "get_table_cql"
	CmdProtoFileReaderCreator string = "proto_file_reader_creator"
	CmdCheckDbConnectivity    string = "check_db_connectivity"
	CmdCheckQueueConnectivity string = "check_queue_connectivity"
)

func usage(flagset *flag.FlagSet) {
	fmt.Printf("Capillaries toolbelt\nUsage: capitoolbelt <command> <command parameters>\nCommands:\n")
	fmt.Printf("  %s\n  %s\n  %s\n  %s\n  %s\n  %s\n  %s\n  %s\n  %s\n  %s\n  %s\n  %s\n  %s\n",
		CmdValidateScript,
		CmdStartRun,
		CmdStopRun,
		CmdExecNode,
		CmdGetRunHistory,
		CmdGetNodeHistory,
		CmdGetBatchHistory,
		CmdGetRunStatusDiagram,
		CmdDropKeyspace,
		CmdGetTableCql,
		CmdProtoFileReaderCreator,
		CmdCheckDbConnectivity,
		CmdCheckQueueConnectivity)
	if flagset != nil {
		fmt.Printf("\n%s parameters:\n", flagset.Name())
		flagset.PrintDefaults()
	}
}

func validateScript(envConfig *env.EnvConfig) int {
	validateScriptCmd := flag.NewFlagSet(CmdValidateScript, flag.ExitOnError)
	scriptFilePath := validateScriptCmd.String("script_file", "", "Path to script file")
	paramsFilePath := validateScriptCmd.String("params_file", "", "Path to script parameters map file")
	isIdxDag := validateScriptCmd.Bool("idx_dag", false, "Print index DAG")
	isFieldDag := validateScriptCmd.Bool("field_dag", false, "Print field DAG")
	if err := validateScriptCmd.Parse(os.Args[2:]); err != nil || *scriptFilePath == "" {
		usage(validateScriptCmd)
		return 0
	}

	script, _, err := sc.NewScriptFromFiles(envConfig.CaPath, envConfig.PrivateKeys, *scriptFilePath, *paramsFilePath, envConfig.CustomProcessorDefFactoryInstance, envConfig.CustomProcessorsSettings)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	if *isIdxDag {
		fmt.Println(getDotDiagram(script, DotDiagramIndexes, nil))
	}
	if *isFieldDag {
		fmt.Println(getDotDiagram(script, DotDiagramFields, nil))
	}
	return 0
}

func startRun(envConfig *env.EnvConfig, logger *l.CapiLogger) int {
	startRunCmd := flag.NewFlagSet(CmdStartRun, flag.ExitOnError)
	keyspace := startRunCmd.String("keyspace", "", "Keyspace (session id)")
	scriptFilePath := startRunCmd.String("script_file", "", "Path to script file")
	paramsFilePath := startRunCmd.String("params_file", "", "Path to script parameters map file")
	startNodesString := startRunCmd.String("start_nodes", "", "Comma-separated list of start node names")
	if err := startRunCmd.Parse(os.Args[2:]); err != nil {
		usage(startRunCmd)
		return 0
	}

	startNodes := strings.Split(*startNodesString, ",")

	cqlSession, err := db.NewSession(envConfig, *keyspace, db.CreateKeyspaceOnConnect)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	// RabbitMQ boilerplate
	amqpConnection, err := amqp.Dial(envConfig.Amqp.URL)
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("cannot dial RabbitMQ at %s, will reconnect: %s\n", envConfig.Amqp.URL, err.Error()))
		return 1
	}
	defer amqpConnection.Close()

	amqpChannel, err := amqpConnection.Channel()
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Sprintf("cannot create amqp channel, will reconnect: %s\n", err.Error()))
		return 1
	}
	defer amqpChannel.Close()

	runId, err := api.StartRun(envConfig, logger, amqpChannel, *scriptFilePath, *paramsFilePath, cqlSession, *keyspace, startNodes, "started by Toolbelt")
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	fmt.Println(runId)
	return 0
}

func stopRun(envConfig *env.EnvConfig, logger *l.CapiLogger) int {
	stopRunCmd := flag.NewFlagSet(CmdStopRun, flag.ExitOnError)
	keyspace := stopRunCmd.String("keyspace", "", "Keyspace (session id)")
	runIdString := stopRunCmd.String("run_id", "", "Run id")
	if err := stopRunCmd.Parse(os.Args[2:]); err != nil {
		usage(stopRunCmd)
		return 0
	}

	runId, err := strconv.ParseInt(strings.TrimSpace(*runIdString), 10, 16)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	cqlSession, err := db.NewSession(envConfig, *keyspace, db.DoNotCreateKeyspaceOnConnect)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	err = api.StopRun(logger, cqlSession, *keyspace, int16(runId), "stopped by toolbelt")
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	return 0
}

func getRunHistory(envConfig *env.EnvConfig, logger *l.CapiLogger) int {
	getRunsCmd := flag.NewFlagSet(CmdGetRunHistory, flag.ExitOnError)
	keyspace := getRunsCmd.String("keyspace", "", "Keyspace (session id)")
	if err := getRunsCmd.Parse(os.Args[2:]); err != nil {
		usage(getRunsCmd)
		return 0
	}

	cqlSession, err := db.NewSession(envConfig, *keyspace, db.DoNotCreateKeyspaceOnConnect)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	runs, err := api.GetRunHistory(logger, cqlSession, *keyspace)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	fmt.Println(strings.Join(wfmodel.RunHistoryEventAllFields(), ","))
	for _, r := range runs {
		fmt.Printf("%s,%d,%d,%s\n", r.Ts.Format(LogTsFormatUnquoted), r.RunId, r.Status, strings.ReplaceAll(r.Comment, ",", ";"))
	}
	return 0
}

func getNodeHistory(envConfig *env.EnvConfig, logger *l.CapiLogger) int {
	getNodeHistoryCmd := flag.NewFlagSet(CmdGetNodeHistory, flag.ExitOnError)
	keyspace := getNodeHistoryCmd.String("keyspace", "", "Keyspace (session id)")
	runIdsString := getNodeHistoryCmd.String("run_ids", "", "Limit results to specific run ids (optional), comma-separated list")
	if err := getNodeHistoryCmd.Parse(os.Args[2:]); err != nil {
		usage(getNodeHistoryCmd)
		return 0
	}

	cqlSession, err := db.NewSession(envConfig, *keyspace, db.DoNotCreateKeyspaceOnConnect)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	runIds, err := stringToArrayOfInt16(*runIdsString)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	nodes, err := api.GetNodeHistoryForRuns(logger, cqlSession, *keyspace, runIds)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	fmt.Println(strings.Join(wfmodel.NodeHistoryEventAllFields(), ","))
	for _, n := range nodes {
		fmt.Printf("%s,%d,%s,%d,%s\n", n.Ts.Format(LogTsFormatUnquoted), n.RunId, n.ScriptNode, n.Status, strings.ReplaceAll(n.Comment, ",", ";"))
	}
	return 0
}

func getBatchHistory(envConfig *env.EnvConfig, logger *l.CapiLogger) int {
	getBatchHistoryCmd := flag.NewFlagSet(CmdGetBatchHistory, flag.ExitOnError)
	keyspace := getBatchHistoryCmd.String("keyspace", "", "Keyspace (session id)")
	runIdsString := getBatchHistoryCmd.String("run_ids", "", "Limit results to specific run ids (optional), comma-separated list")
	nodeNamesString := getBatchHistoryCmd.String("nodes", "", "Limit results to specific node names (optional), comma-separated list")
	if err := getBatchHistoryCmd.Parse(os.Args[2:]); err != nil {
		usage(getBatchHistoryCmd)
		return 0
	}

	cqlSession, err := db.NewSession(envConfig, *keyspace, db.DoNotCreateKeyspaceOnConnect)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	runIds, err := stringToArrayOfInt16(*runIdsString)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	var nodeNames []string
	if len(strings.TrimSpace(*nodeNamesString)) > 0 {
		nodeNames = strings.Split(*nodeNamesString, ",")
	} else {
		nodeNames = make([]string, 0)
	}

	runs, err := api.GetBatchHistoryForRunsAndNodes(logger, cqlSession, *keyspace, runIds, nodeNames)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	fmt.Println(strings.Join(wfmodel.BatchHistoryEventAllFields(), ","))
	for _, r := range runs {
		fmt.Printf("%s,%d,%s,%d,%d,%d,%d,%d,%s\n", r.Ts.Format(LogTsFormatUnquoted), r.RunId, r.ScriptNode, r.BatchIdx, r.BatchesTotal, r.Status, r.FirstToken, r.LastToken, strings.ReplaceAll(r.Comment, ",", ";"))
	}
	return 0
}

func getTableCql(envConfig *env.EnvConfig) int {
	getTableCqlCmd := flag.NewFlagSet(CmdGetTableCql, flag.ExitOnError)
	scriptFilePath := getTableCqlCmd.String("script_file", "", "Path to script file")
	paramsFilePath := getTableCqlCmd.String("params_file", "", "Path to script parameters map file")
	keyspace := getTableCqlCmd.String("keyspace", "", "Keyspace (session id)")
	runId := getTableCqlCmd.Int("run_id", 0, "Run id")
	startNodesString := getTableCqlCmd.String("start_nodes", "", "Comma-separated list of start node names")
	if err := getTableCqlCmd.Parse(os.Args[2:]); err != nil {
		usage(getTableCqlCmd)
		return 0
	}

	script, _, err := sc.NewScriptFromFiles(envConfig.CaPath, envConfig.PrivateKeys, *scriptFilePath, *paramsFilePath, envConfig.CustomProcessorDefFactoryInstance, envConfig.CustomProcessorsSettings)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	startNodes := strings.Split(*startNodesString, ",")
	fmt.Print(api.GetTablesCql(script, *keyspace, int16(*runId), startNodes))
	return 0
}

func getRunStatusDiagram(envConfig *env.EnvConfig, logger *l.CapiLogger) int {
	getRunStatusDiagramCmd := flag.NewFlagSet(CmdGetRunStatusDiagram, flag.ExitOnError)
	scriptFilePath := getRunStatusDiagramCmd.String("script_file", "", "Path to script file")
	paramsFilePath := getRunStatusDiagramCmd.String("params_file", "", "Path to script parameters map file")
	keyspace := getRunStatusDiagramCmd.String("keyspace", "", "Keyspace (session id)")
	runIdString := getRunStatusDiagramCmd.String("run_id", "", "Run id")
	if err := getRunStatusDiagramCmd.Parse(os.Args[2:]); err != nil {
		usage(getRunStatusDiagramCmd)
		return 0
	}

	runId, err := strconv.ParseInt(strings.TrimSpace(*runIdString), 10, 16)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	script, _, err := sc.NewScriptFromFiles(envConfig.CaPath, envConfig.PrivateKeys, *scriptFilePath, *paramsFilePath, envConfig.CustomProcessorDefFactoryInstance, envConfig.CustomProcessorsSettings)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	cqlSession, err := db.NewSession(envConfig, *keyspace, db.DoNotCreateKeyspaceOnConnect)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	nodes, err := api.GetNodeHistoryForRuns(logger, cqlSession, *keyspace, []int16{int16(runId)})
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	nodeColorMap := map[string]string{}
	for _, node := range nodes {
		nodeColorMap[node.ScriptNode] = nodeBatchStatusToColor(node.Status)
	}

	fmt.Println(getDotDiagram(script, DotDiagramRunStatus, nodeColorMap))
	return 0
}

func dropKeyspace(envConfig *env.EnvConfig, logger *l.CapiLogger) int {
	dropKsCmd := flag.NewFlagSet(CmdDropKeyspace, flag.ExitOnError)
	keyspace := dropKsCmd.String("keyspace", "", "Keyspace (session id)")
	if err := dropKsCmd.Parse(os.Args[2:]); err != nil {
		usage(dropKsCmd)
		return 0
	}

	cqlSession, err := db.NewSession(envConfig, *keyspace, db.DoNotCreateKeyspaceOnConnect)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	err = api.DropKeyspace(logger, cqlSession, *keyspace)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	return 0
}

func checkDbConnectivity(envConfig *env.EnvConfig) int {
	cqlSession, err := db.NewSession(envConfig, "", db.CreateKeyspaceOnConnect)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	cqlSession.Close()
	fmt.Fprintf(os.Stdout, "OK: %v\n", envConfig.Cassandra.Hosts)
	return 0
}

func checkQueueConnectivity(envConfig *env.EnvConfig) int {
	amqpConnection, err := amqp.Dial(envConfig.Amqp.URL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot dial RabbitMQ at %s: %s\n", envConfig.Amqp.URL, err.Error())
		return 1
	}
	amqpConnection.Close()
	fmt.Fprintf(os.Stdout, "OK: %s\n", envConfig.Amqp.URL)
	return 0
}

func execNode(envConfig *env.EnvConfig, logger *l.CapiLogger) int {
	execNodeCmd := flag.NewFlagSet(CmdExecNode, flag.ExitOnError)
	keyspace := execNodeCmd.String("keyspace", "", "Keyspace (session id)")
	scriptFilePath := execNodeCmd.String("script_file", "", "Path to script file")
	paramsFilePath := execNodeCmd.String("params_file", "", "Path to script parameters map file")
	runIdParam := execNodeCmd.Int("run_id", 0, "run id (optional, use with extra caution as it will modify existing run id results)")
	nodeName := execNodeCmd.String("node_id", "", "Script node name")
	if err := execNodeCmd.Parse(os.Args[2:]); err != nil {
		usage(execNodeCmd)
		return 0
	}

	runId := int16(*runIdParam)

	startTime := time.Now()

	cqlSession, err := db.NewSession(envConfig, *keyspace, db.CreateKeyspaceOnConnect)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}

	runId, err = api.RunNode(envConfig, logger, *nodeName, runId, *scriptFilePath, *paramsFilePath, cqlSession, *keyspace)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	fmt.Printf("run %d, elapsed %v\n", runId, time.Since(startTime))
	return 0
}

func protoFileReaderCreator() int {
	cmd := flag.NewFlagSet(CmdProtoFileReaderCreator, flag.ExitOnError)
	filePath := cmd.String("file", "", "path to sample file")
	fileType := cmd.String("file_type", "csv", "csv or parquet")
	csvHeaderLine := cmd.Int("csv_hdr_line_idx", -1, "csv only: index of the header line")
	csvFirstDataLine := cmd.Int("csv_first_line_idx", 0, "csv only: index of the first data line, must be greater than csv_hdr_line_idx")
	csvSeparator := cmd.String("csv_separator", ",", "csv only: field separator")
	if err := cmd.Parse(os.Args[2:]); err != nil || *filePath == "" || (*fileType != "csv" && *fileType != "parquet") || *csvHeaderLine >= *csvFirstDataLine || *csvSeparator == "" {
		usage(cmd)
		return 0
	}

	srcFileName := filepath.Base(*filePath)

	srcFileFinalPath := "/tmp/capi_in/proto_file_reader_creator_quicktest/" + srcFileName
	tgtFileFinalPath := "/tmp/capi_out/proto_file_reader_creator_quicktest/" + srcFileName

	// Some reasonable defaults that match our test expectations in test/data/proto_file_reader_creator
	fileWriterFormatMap := map[sc.TableFieldType]string{
		sc.FieldTypeString:   "%s",
		sc.FieldTypeInt:      "%d",
		sc.FieldTypeFloat:    "%.3f",
		sc.FieldTypeDecimal2: "%s",
		sc.FieldTypeDateTime: "2006-01-02 15:04:05",
		sc.FieldTypeBool:     "%t"}

	// Create file reader
	fileReaderDef := sc.FileReaderDef{
		SrcFileUrls: []string{srcFileFinalPath},
		Columns:     map[string]*sc.FileReaderColumnDef{}}

	// Create file creator
	fileCreatorDef := sc.FileCreatorDef{
		UrlTemplate: tgtFileFinalPath}

	var errGuess error
	var guessedFields []*storage.GuessedField
	var fieldSettingsRemover *regexp.Regexp
	if *fileType == "csv" {
		guessedFields, errGuess = storage.CsvGuessFields(*filePath, *csvHeaderLine, *csvFirstDataLine, *csvSeparator)
		if errGuess != nil && (guessedFields == nil || len(guessedFields) == 0) {
			fmt.Fprintln(os.Stderr, errGuess.Error())
			return 1
		}
		fileReaderDef.ReaderFileType = sc.ReaderFileTypeCsv
		fileReaderDef.Csv = sc.CsvReaderSettings{
			SrcFileHdrLineIdx:       *csvHeaderLine,
			SrcFileFirstDataLineIdx: *csvFirstDataLine,
			Separator:               *csvSeparator,
			ColumnIndexingMode:      sc.FileColumnIndexingUnknown}

		for colIdx, gf := range guessedFields {
			colIdxToUse := colIdx
			if len(gf.OriginalHeader) > 0 {
				// Do not use col indexes, use headers
				colIdxToUse = 0
			}
			fileReaderDef.Columns[gf.CapiName] = &sc.FileReaderColumnDef{
				Type: gf.Type,
				Csv: sc.CsvReaderColumnSettings{
					SrcColIdx:    colIdxToUse,
					SrcColHeader: gf.OriginalHeader,
					SrcColFormat: gf.Format}}
		}
		fileCreatorDef.Csv.Separator = *csvSeparator
		fileCreatorDef.Columns = make([]sc.WriteFileColumnDef, len(guessedFields))
		for colIdx, gf := range guessedFields {
			fileCreatorDef.Columns[colIdx] = sc.WriteFileColumnDef{
				Name:          gf.CapiName,
				RawExpression: "r." + strings.ReplaceAll(gf.CapiName, "col_", ""),
				Type:          gf.Type,
				Csv: sc.WriteCsvColumnSettings{
					Format: fileWriterFormatMap[gf.Type],
					Header: gf.OriginalHeader}}
		}
		fieldSettingsRemover = regexp.MustCompile(`,*[ \t\n]*"parquet":[ \t\n]*{[^}]*}`)
	} else {
		guessedFields, errGuess = storage.ParquetGuessFields(*filePath)
		if errGuess != nil && (guessedFields == nil || len(guessedFields) == 0) {
			fmt.Fprintln(os.Stderr, errGuess.Error())
			return 1
		}
		fileReaderDef.ReaderFileType = sc.ReaderFileTypeParquet
		for _, gf := range guessedFields {
			fileReaderDef.Columns[gf.CapiName] = &sc.FileReaderColumnDef{
				Type: gf.Type,
				Parquet: sc.ParquetReaderColumnSettings{
					SrcColName: gf.OriginalHeader}}
		}
		fileCreatorDef.Parquet.Codec = sc.ParquetCodecGzip
		fileCreatorDef.Columns = make([]sc.WriteFileColumnDef, len(guessedFields))
		for colIdx, gf := range guessedFields {
			fileCreatorDef.Columns[colIdx] = sc.WriteFileColumnDef{
				Name:          gf.CapiName,
				RawExpression: "r." + strings.ReplaceAll(gf.CapiName, "col_", ""),
				Type:          gf.Type,
				Parquet: sc.WriteParquetColumnSettings{
					ColumnName: gf.OriginalHeader}}
		}
		fieldSettingsRemover = regexp.MustCompile(`[ \t\n]*"csv":[ \t\n]*{[^}]*},*`)
	}

	reNonAlphanum := regexp.MustCompile("[^a-zA-Z0-9_]")
	sanitizedFileName := reNonAlphanum.ReplaceAllString(srcFileName, "_")

	// Create matching table creator
	tableCreatorDef := sc.TableCreatorDef{
		Name:   sanitizedFileName,
		Fields: map[string]*sc.WriteTableFieldDef{}}

	for capiFileColName, readerColDef := range fileReaderDef.Columns {
		capiTableFieldName := strings.Replace(capiFileColName, "col_", "", 1)
		tableCreatorDef.Fields[capiTableFieldName] = &sc.WriteTableFieldDef{
			RawExpression: "r." + capiFileColName,
			Type:          readerColDef.Type}
	}

	// Create matching table reader
	tableReaderDef := sc.TableReaderDef{
		TableName: sanitizedFileName}

	fileReaderBytes, errMarshal := json.Marshal(fileReaderDef)
	if errMarshal != nil {
		fmt.Fprintln(os.Stderr, errMarshal.Error())
		return 1
	}

	tableWriterBytes, errMarshal := json.Marshal(tableCreatorDef)
	if errMarshal != nil {
		fmt.Fprintln(os.Stderr, errMarshal.Error())
		return 1
	}

	tableReaderBytes, errMarshal := json.Marshal(tableReaderDef)
	if errMarshal != nil {
		fmt.Fprintln(os.Stderr, errMarshal.Error())
		return 1
	}

	fileCreatorBytes, errMarshal := json.Marshal(fileCreatorDef)
	if errMarshal != nil {
		fmt.Fprintln(os.Stderr, errMarshal.Error())
		return 1
	}

	topRemover := regexp.MustCompile(`[ \t\n]*"top":[ \t\n]*{[^}]*}[ \t\n]*},*`)

	finalJson := fmt.Sprintf(
		`{
 "nodes": {
  "read_file": {
   "type": "file_table",
   "desc": "Read file %s to table",
   "start_policy": "manual",
   "r": %s,
   "w": %s
  },
  "write_file": {
	"type": "table_file",
	"desc": "Write from table to file %s",
	"r": %s,
	"w": %s
  }
 }, 
 "dependency_policies": {
  "current_active_first_stopped_nogo": %s
  }
}`,
		srcFileFinalPath,
		fieldSettingsRemover.ReplaceAllString(string(fileReaderBytes), ""),
		string(tableWriterBytes),
		tgtFileFinalPath,
		string(tableReaderBytes),
		topRemover.ReplaceAllString(fieldSettingsRemover.ReplaceAllString(string(fileCreatorBytes), ""), ""),
		sc.DefaultPolicyCheckerConf)

	finalJsonBytes := bytes.Buffer{}
	errMarshal = json.Indent(&finalJsonBytes, []byte(finalJson), "", " ")
	if errMarshal != nil {
		fmt.Fprintln(os.Stderr, errMarshal.Error())
		return 1
	}

	fmt.Printf("%s", finalJsonBytes.String())

	if errGuess != nil {
		fmt.Fprintln(os.Stderr, errGuess.Error())
		return 1
	}

	return 0
}

func main() {
	// defer profile.Start().Stop()

	initCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	envConfig, err := env.ReadEnvConfigFile(initCtx, "capitoolbelt.json")
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	envConfig.CustomProcessorDefFactoryInstance = &StandardToolbeltProcessorDefFactory{}
	logger, err := l.NewLoggerFromEnvConfig(envConfig)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	defer logger.Close()

	if len(os.Args) <= 1 {
		usage(nil)
		os.Exit(0)
	}

	switch os.Args[1] {

	case CmdValidateScript:
		os.Exit(validateScript(envConfig))

	case CmdStartRun:
		os.Exit(startRun(envConfig, logger))

	case CmdStopRun:
		os.Exit(stopRun(envConfig, logger))

	case CmdGetRunHistory:
		os.Exit(getRunHistory(envConfig, logger))

	case CmdGetNodeHistory:
		os.Exit(getNodeHistory(envConfig, logger))

	case CmdGetBatchHistory:
		os.Exit(getBatchHistory(envConfig, logger))

	case CmdGetTableCql:
		os.Exit(getTableCql(envConfig))

	case CmdGetRunStatusDiagram:
		os.Exit(getRunStatusDiagram(envConfig, logger))

	case CmdDropKeyspace:
		os.Exit(dropKeyspace(envConfig, logger))

	case CmdExecNode:
		os.Exit(execNode(envConfig, logger))

	case CmdProtoFileReaderCreator:
		os.Exit(protoFileReaderCreator())

	case CmdCheckDbConnectivity:
		os.Exit(checkDbConnectivity(envConfig))

	case CmdCheckQueueConnectivity:
		os.Exit(checkQueueConnectivity(envConfig))

	default:
		fmt.Printf("invalid command: %s\n", os.Args[1])
		usage(nil)
		os.Exit(1)
	}
}
