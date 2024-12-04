package api

import (
	"fmt"
	"strings"

	"github.com/capillariesio/capillaries/pkg/sc"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
)

type DiagramType string

const (
	DiagramIndexes   DiagramType = "indexes"
	DiagramFields    DiagramType = "fields"
	DiagramRunStatus DiagramType = "run_status"
)

func NodeBatchStatusToDotColor(status wfmodel.NodeBatchStatusType) string {
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

func drawFileReader(node *sc.ScriptNodeDef, dotDiagramType DiagramType, arrowFontSize int, recordFontSize int) string {
	var b strings.Builder
	arrowLabelBuilder := strings.Builder{}
	if dotDiagramType == DiagramType(DiagramFields) {
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

func drawFileCreator(node *sc.ScriptNodeDef, dotDiagramType DiagramType, arrowFontSize int, recordFontSize int, allUsedFields sc.FieldRefs, penWidth string, fillColor string, urlEscaper *strings.Replacer, inSrcArrowLabel string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("\"%s\" -> \"%s\" [style=solid, fontsize=\"%d\", label=\"%s\"];\n", node.TableReader.TableName, node.Name, arrowFontSize, inSrcArrowLabel))

	// Node (file)
	b.WriteString(fmt.Sprintf("\"%s\" [shape=record, penwidth=\"%s\", fontsize=\"%d\", fillcolor=\"%s\", style=\"filled\", label=\"{%s|creates file:\\n%s}\", tooltip=\"%s\"];\n", node.Name, penWidth, recordFontSize, fillColor, node.Name, urlEscaper.Replace(node.FileCreator.UrlTemplate), node.Desc))

	// Out (file)
	arrowLabelBuilder := strings.Builder{}
	if dotDiagramType == DiagramType(DiagramFields) {
		for i := 0; i < len(allUsedFields); i++ {
			arrowLabelBuilder.WriteString(allUsedFields[i].FieldName)
			arrowLabelBuilder.WriteString("\\l")
		}
	}

	b.WriteString(fmt.Sprintf("\"%s\" -> \"%s\" [style=dotted, fontsize=\"%d\", label=\"%s\"];\n", node.Name, node.FileCreator.UrlTemplate, arrowFontSize, arrowLabelBuilder.String()))
	b.WriteString(fmt.Sprintf("\"%s\" [shape=note, fontsize=\"%d\", label=\"%s\", tooltip=\"Target data file(s)\"];\n", node.FileCreator.UrlTemplate, recordFontSize, node.FileCreator.UrlTemplate))
	return b.String()
}

func drawTableReader(node *sc.ScriptNodeDef, dotDiagramType DiagramType, arrowFontSize int, recordFontSize int, allUsedFields sc.FieldRefs, penWidth string, fillColor string, urlEscaper *strings.Replacer) string {
	var b strings.Builder
	var inSrcArrowLabel string
	if dotDiagramType == DiagramType(DiagramIndexes) || dotDiagramType == DiagramType(DiagramRunStatus) {
		if node.TableReader.ExpectedBatchesTotal > 1 {
			inSrcArrowLabel = fmt.Sprintf("%s (%d batches)", node.TableReader.TableName, node.TableReader.ExpectedBatchesTotal)
		} else {
			inSrcArrowLabel = fmt.Sprintf("%s (no parallelism)", node.TableReader.TableName)
		}
	} else if dotDiagramType == DiagramType(DiagramFields) {
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
		if dotDiagramType == DiagramType(DiagramFields) {
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

// Used by Toolbelt and Webapi
func GetDotDiagram(scriptDef *sc.ScriptDef, diagramType DiagramType, nodeColorMap map[string]string) string {
	var b strings.Builder

	const recordFontSize int = 20
	const arrowFontSize int = 18

	urlEscaper := strings.NewReplacer(`{`, `\{`, `}`, `\}`, `|`, `\|`)
	b.WriteString(fmt.Sprintf("\ndigraph %s {\nrankdir=\"TD\";\n node [fontname=\"Helvetica\"];\nedge [fontname=\"Helvetica\"];\ngraph [splines=true, pad=\"0.5\", ranksep=\"0.5\", nodesep=\"0.5\"];\n", diagramType))
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
			b.WriteString(drawFileReader(node, diagramType, arrowFontSize, recordFontSize))
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
			b.WriteString(drawTableReader(node, diagramType, arrowFontSize, recordFontSize, allUsedFields, penWidth, fillColor, urlEscaper))
		}

		if node.HasTableCreator() {
			b.WriteString(drawTableCreator(node, recordFontSize, penWidth, fillColor))
		}
	}
	b.WriteString("}\n")

	return b.String()
}
