package sc

import (
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"math"
	"regexp"
	"strings"

	"github.com/capillariesio/capillaries/pkg/eval"
)

const (
	HandlerExeTypeGeneric  string = "capi_daemon"
	HandlerExeTypeToolbelt string = "capi_toolbelt"
	HandlerExeTypeWebapi   string = "capi_webapi"
)

const MaxAcceptedBatchesByTableReader int = 1000000
const DefaultRowsetSize int = 1000 // 1000 seems to work on c7g.large without OOM, careful with using bigger values
const MaxRowsetSize int = 100000

type AggFinderVisitor struct {
	Error error
}

func (v *AggFinderVisitor) Visit(node ast.Node) ast.Visitor {
	switch callExp := node.(type) {
	case *ast.CallExpr:
		switch callIdentExp := callExp.Fun.(type) {
		case *ast.Ident:
			if eval.StringToAggFunc(callIdentExp.Name) != eval.AggUnknown {
				v.Error = fmt.Errorf("found aggregate function %s()", callIdentExp.Name)
				return nil
			}
			return v
		default:
			return v
		}
	default:
		return v
	}
}

type NodeType string

const (
	NodeTypeNone                NodeType = "none"
	NodeTypeFileTable           NodeType = "file_table"
	NodeTypeTableTable          NodeType = "table_table"
	NodeTypeTableLookupTable    NodeType = "table_lookup_table"
	NodeTypeTableFile           NodeType = "table_file"
	NodeTypeTableCustomTfmTable NodeType = "table_custom_tfm_table"
	NodeTypeDistinctTable       NodeType = "distinct_table"
)

func ValidateNodeType(nodeType NodeType) error {
	if nodeType == NodeTypeFileTable ||
		nodeType == NodeTypeTableTable ||
		nodeType == NodeTypeTableLookupTable ||
		nodeType == NodeTypeTableFile ||
		nodeType == NodeTypeDistinctTable ||
		nodeType == NodeTypeTableCustomTfmTable {
		return nil
	}
	return fmt.Errorf("invalid node type %s", nodeType)
}

const ReaderAlias string = "r"
const CreatorAlias string = "w"
const LookupAlias string = "l"
const CustomProcessorAlias string = "p"

type NodeRerunPolicy string

const (
	NodeRerun NodeRerunPolicy = "rerun" // Default
	NodeFail  NodeRerunPolicy = "fail"
)

func ValidateRerunPolicy(rerunPolicy NodeRerunPolicy) error {
	if rerunPolicy == NodeRerun ||
		rerunPolicy == NodeFail {
		return nil
	}
	return fmt.Errorf("invalid node rerun policy %s", rerunPolicy)
}

type NodeStartPolicy string

const (
	NodeStartManual NodeStartPolicy = "manual"
	NodeStartAuto   NodeStartPolicy = "auto" // Default
)

func ValidateStartPolicy(startPolicy NodeStartPolicy) error {
	if startPolicy == NodeStartManual ||
		startPolicy == NodeStartAuto {
		return nil
	}
	return fmt.Errorf("invalid node start policy %s", startPolicy)
}

type ScriptNodeDef struct {
	Name                string          // Get it from the key
	Type                NodeType        `json:"type" yaml:"type"`
	Desc                string          `json:"desc" yaml:"desc"`
	StartPolicy         NodeStartPolicy `json:"start_policy" yaml:"start_policy"`
	RerunPolicy         NodeRerunPolicy `json:"rerun_policy,omitempty" yaml:"rerun_policy,omitempty"`
	CustomProcessorType string          `json:"custom_proc_type,omitempty" yaml:"custom_proc_type,omitempty"`
	HandlerExeType      string          `json:"handler_exe_type,omitempty" yaml:"handler_exe_type,omitempty"`

	RawReader   json.RawMessage `json:"r" yaml:"r"` // This depends on tfm type
	TableReader TableReaderDef
	FileReader  FileReaderDef

	Lookup LookupDef `json:"l" yaml:"l"`

	RawProcessorDef json.RawMessage    `json:"p" yaml:"p"` // This depends on tfm type
	CustomProcessor CustomProcessorDef // Also should implement CustomProcessorRunner

	RawWriter            json.RawMessage `json:"w" yaml:"w"` // This depends on tfm type
	DependencyPolicyName string          `json:"dependency_policy" yaml:"dependency_policy"`
	TableCreator         TableCreatorDef
	TableUpdater         TableUpdaterDef
	FileCreator          FileCreatorDef
	DepPolDef            *DependencyPolicyDef
}

func (node *ScriptNodeDef) HasTableReader() bool {
	return node.Type == NodeTypeTableTable ||
		node.Type == NodeTypeTableLookupTable ||
		node.Type == NodeTypeTableFile ||
		node.Type == NodeTypeDistinctTable ||
		node.Type == NodeTypeTableCustomTfmTable
}
func (node *ScriptNodeDef) HasFileReader() bool {
	return node.Type == NodeTypeFileTable
}

func (node *ScriptNodeDef) HasLookup() bool {
	return node.Type == NodeTypeTableLookupTable
}

func (node *ScriptNodeDef) HasCustomProcessor() bool {
	return node.Type == NodeTypeTableCustomTfmTable
}

func (node *ScriptNodeDef) HasTableCreator() bool {
	return node.Type == NodeTypeFileTable ||
		node.Type == NodeTypeTableTable ||
		node.Type == NodeTypeDistinctTable ||
		node.Type == NodeTypeTableLookupTable ||
		node.Type == NodeTypeTableCustomTfmTable
}
func (node *ScriptNodeDef) HasFileCreator() bool {
	return node.Type == NodeTypeTableFile
}
func (node *ScriptNodeDef) GetTargetName() string {
	if node.HasTableCreator() {
		return node.TableCreator.Name
	} else if node.HasFileCreator() {
		return CreatorAlias
	}
	return "dev_error_uknown_target_name"
}

func (node *ScriptNodeDef) initReader() error {
	if node.HasTableReader() {
		if err := json.Unmarshal(node.RawReader, &node.TableReader); err != nil {
			return fmt.Errorf("cannot unmarshal table reader: [%s]", err.Error())
		}
		foundErrors := make([]string, 0)
		if len(node.TableReader.TableName) == 0 {
			foundErrors = append(foundErrors, "table reader cannot reference empty table name")
		}
		if node.TableReader.ExpectedBatchesTotal == 0 {
			node.TableReader.ExpectedBatchesTotal = 1
		} else if node.TableReader.ExpectedBatchesTotal < 0 || node.TableReader.ExpectedBatchesTotal > MaxAcceptedBatchesByTableReader {
			foundErrors = append(foundErrors, fmt.Sprintf("table reader can accept between 1 and %d batches, %d specified", MaxAcceptedBatchesByTableReader, node.TableReader.ExpectedBatchesTotal))
		}
		if node.TableReader.RowsetSize < 0 || MaxRowsetSize < node.TableReader.RowsetSize {
			foundErrors = append(foundErrors, fmt.Sprintf("invalid rowset size %d, table reader can accept between 0 (defaults to %d) and %d", node.TableReader.RowsetSize, DefaultRowsetSize, MaxRowsetSize))
		}
		if node.TableReader.RowsetSize == 0 {
			node.TableReader.RowsetSize = DefaultRowsetSize
		}
		if len(foundErrors) > 0 {
			return fmt.Errorf("%s", strings.Join(foundErrors, "; "))
		}
	} else if node.HasFileReader() {
		if err := node.FileReader.Deserialize(node.RawReader); err != nil {
			return fmt.Errorf("cannot deserialize file reader [%s]: [%s]", string(node.RawReader), err.Error())
		}
	}

	return nil
}

func (node *ScriptNodeDef) initCreator() error {
	if node.HasTableCreator() {
		if err := node.TableCreator.Deserialize(node.RawWriter); err != nil {
			return fmt.Errorf("cannot deserialize table creator [%s]: [%s]", strings.ReplaceAll(string(node.RawWriter), "\n", " "), err.Error())
		}
	} else if node.HasFileCreator() {
		if err := node.FileCreator.Deserialize(node.RawWriter); err != nil {
			return fmt.Errorf("cannot deserialize file creator [%s]: [%s]", strings.ReplaceAll(string(node.RawWriter), "\n", " "), err.Error())
		}
	}
	return nil
}

func (node *ScriptNodeDef) initCustomProcessor(customProcessorDefFactory CustomProcessorDefFactory, customProcessorsSettings map[string]json.RawMessage, scriptType ScriptType, caPath string, privateKeys map[string]string) error {
	if node.HasCustomProcessor() {
		if customProcessorDefFactory == nil {
			return errors.New("undefined custom processor factory")
		}
		if customProcessorsSettings == nil {
			return errors.New("missing custom processor settings section")
		}
		var ok bool
		node.CustomProcessor, ok = customProcessorDefFactory.Create(node.CustomProcessorType)
		if !ok {
			return fmt.Errorf("cannot deserialize unknown custom processor %s", node.CustomProcessorType)
		}
		customProcSettings, ok := customProcessorsSettings[node.CustomProcessorType]
		if !ok {
			return fmt.Errorf("cannot find custom processing settings for [%s] in the environment config file", node.CustomProcessorType)
		}
		if err := node.CustomProcessor.Deserialize(node.RawProcessorDef, customProcSettings, scriptType, caPath, privateKeys); err != nil {
			re := regexp.MustCompile("[ \r\n]+")
			return fmt.Errorf("cannot deserialize custom processor [%s]: [%s]", re.ReplaceAllString(string(node.RawProcessorDef), ""), err.Error())
		}
	}
	return nil
}

func (node *ScriptNodeDef) Deserialize(customProcessorDefFactory CustomProcessorDefFactory, customProcessorsSettings map[string]json.RawMessage, scriptType ScriptType, caPath string, privateKeys map[string]string) error {
	foundErrors := make([]string, 0)

	if err := ValidateNodeType(node.Type); err != nil {
		return err
	}

	// Defaults

	if len(node.HandlerExeType) == 0 {
		node.HandlerExeType = HandlerExeTypeGeneric
	}

	if len(node.RerunPolicy) == 0 {
		node.RerunPolicy = NodeRerun
	} else if err := ValidateRerunPolicy(node.RerunPolicy); err != nil {
		return err
	}

	if len(node.StartPolicy) == 0 {
		node.StartPolicy = NodeStartAuto
	} else if err := ValidateStartPolicy(node.StartPolicy); err != nil {
		return err
	}

	// Reader
	if err := node.initReader(); err != nil {
		foundErrors = append(foundErrors, err.Error())
	}

	// Creator
	if err := node.initCreator(); err != nil {
		foundErrors = append(foundErrors, err.Error())
	}

	// Custom processor
	if err := node.initCustomProcessor(customProcessorDefFactory, customProcessorsSettings, scriptType, caPath, privateKeys); err != nil {
		foundErrors = append(foundErrors, err.Error())
	}

	// Distinct table
	if node.Type == NodeTypeDistinctTable {
		if node.RerunPolicy != NodeFail {
			foundErrors = append(foundErrors, "distinct_table node must have fail policy, no reruns possible")
		}
		if _, _, err := node.TableCreator.GetSingleUniqueIndexDef(); err != nil {
			foundErrors = append(foundErrors, err.Error())
		}
	}

	if len(foundErrors) > 0 {
		return fmt.Errorf("%s", strings.Join(foundErrors, "; "))
	}

	return nil
}

func (node *ScriptNodeDef) evalCreatorAndLookupExpressionsAndCheckType() error {
	foundErrors := make([]string, 0, 2)

	if node.HasLookup() && node.Lookup.UsesFilter() {
		if err := evalExpressionWithFieldRefsAndCheckType(node.Lookup.Filter, node.Lookup.UsedInFilterFields, FieldTypeBool); err != nil {
			foundErrors = append(foundErrors, fmt.Sprintf("cannot evaluate lookup filter expression [%s]: [%s]", node.Lookup.RawFilter, err.Error()))
		}
	}

	if node.HasTableCreator() {
		// Having
		if err := evalExpressionWithFieldRefsAndCheckType(node.TableCreator.Having, node.TableCreator.UsedInHavingFields, FieldTypeBool); err != nil {
			foundErrors = append(foundErrors, fmt.Sprintf("cannot evaluate table creator 'having' expression [%s]: [%s]", node.TableCreator.RawHaving, err.Error()))
		}

		// Target table fields
		for tgtFieldName, tgtFieldDef := range node.TableCreator.Fields {

			// TODO: find a way to check field usage:
			// - lookup fields must be used only within enclosing agg calls (sum etc), otherwise last one wins
			// - src table fields are allowed within enclosing agg calls, and there is even a biz case for it (multiply src field by the number of lookup rows)

			// If no grouping is used, no agg calls allowed
			if node.HasLookup() && !node.Lookup.IsGroup || !node.HasLookup() {
				v := AggFinderVisitor{}
				if tgtFieldDef.ParsedExpression == nil {
					foundErrors = append(foundErrors, fmt.Sprintf("cannot parse node %s, target field %s expression [%s] was not parsed successfully", node.Name, tgtFieldName, tgtFieldDef.RawExpression))
				} else {
					ast.Walk(&v, tgtFieldDef.ParsedExpression)
					if v.Error != nil {
						foundErrors = append(foundErrors, fmt.Sprintf("cannot use agg functions in [%s], lookup group flag is not set or no lookups used: [%s]", tgtFieldDef.RawExpression, v.Error.Error()))
					}
				}
			}

			// Just eval with test values, agg functions will go through preserving the type no problem
			if err := evalExpressionWithFieldRefsAndCheckType(tgtFieldDef.ParsedExpression, node.TableCreator.UsedInTargetExpressionsFields, tgtFieldDef.Type); err != nil {
				foundErrors = append(foundErrors, fmt.Sprintf("cannot evaluate table creator target field %s expression [%s]: [%s]", tgtFieldName, tgtFieldDef.RawExpression, err.Error()))
			}
		}
	}

	if node.HasFileCreator() {
		// Having
		if err := evalExpressionWithFieldRefsAndCheckType(node.FileCreator.Having, node.FileCreator.UsedInHavingFields, FieldTypeBool); err != nil {
			foundErrors = append(foundErrors, fmt.Sprintf("cannot evaluate file creator 'having' expression [%s]: [%s]", node.FileCreator.RawHaving, err.Error()))
		}

		// Target table fields (yes, they are not just strings, we check the type)
		for i := 0; i < len(node.FileCreator.Columns); i++ {
			colDef := &node.FileCreator.Columns[i]
			if err := evalExpressionWithFieldRefsAndCheckType(colDef.ParsedExpression, node.FileCreator.UsedInTargetExpressionsFields, colDef.Type); err != nil {
				foundErrors = append(foundErrors, fmt.Sprintf("cannot evaluate file creator target field %s expression [%s]: [%s]", colDef.Name, colDef.RawExpression, err.Error()))
			}
		}
	}

	// NOTE: do not even try to eval expressions from the custom processor here,
	// they may contain custom stuff and are pretty much guaranteed to fail

	if len(foundErrors) > 0 {
		return fmt.Errorf("%s", strings.Join(foundErrors, "; "))
	}

	return nil
}

func (node *ScriptNodeDef) getSourceFieldRefs() (*FieldRefs, error) {
	if node.HasFileReader() {
		return node.FileReader.getFieldRefs(), nil
	} else if node.HasTableReader() {
		return node.TableReader.TableCreator.GetFieldRefsWithAlias(ReaderAlias), nil
	}

	return nil, fmt.Errorf("dev error, node of type %s has no file or table reader", node.Type)
}

func (node *ScriptNodeDef) GetUniqueIndexesFieldRefs() *FieldRefs {
	if !node.HasTableCreator() {
		return &FieldRefs{}
	}
	fieldTypeMap := map[string]TableFieldType{}
	for _, idxDef := range node.TableCreator.Indexes {
		if idxDef.Uniqueness == IdxUnique {
			for _, idxComponentDef := range idxDef.Components {
				fieldTypeMap[idxComponentDef.FieldName] = idxComponentDef.FieldType
			}
		}
	}
	fieldRefs := make(FieldRefs, len(fieldTypeMap))
	fieldRefIdx := 0
	for fieldName, fieldType := range fieldTypeMap {
		fieldRefs[fieldRefIdx] = FieldRef{
			FieldName: fieldName,
			FieldType: fieldType,
			TableName: node.TableCreator.Name}
		fieldRefIdx++
	}

	return &fieldRefs
}

func (node *ScriptNodeDef) GetTokenIntervalsByNumberOfBatches() ([][]int64, error) {
	if node.HasTableReader() || node.HasFileCreator() && node.TableReader.ExpectedBatchesTotal > 1 {
		if node.TableReader.ExpectedBatchesTotal == 1 {
			return [][]int64{{int64(math.MinInt64), int64(math.MaxInt64)}}, nil
		}

		tokenIntervalPerBatch := int64(math.MaxInt64/node.TableReader.ExpectedBatchesTotal) - int64(math.MinInt64/node.TableReader.ExpectedBatchesTotal)

		intervals := make([][]int64, node.TableReader.ExpectedBatchesTotal)
		left := int64(math.MinInt64)
		for i := 0; i < len(intervals); i++ {
			var right int64
			if i == len(intervals)-1 {
				right = math.MaxInt64
			} else {
				right = left + tokenIntervalPerBatch - 1
			}
			intervals[i] = []int64{left, right}
			left = right + 1
		}
		return intervals, nil
		// } else if node.HasFileCreator() && node.TableReader.ExpectedBatchesTotal == 1 {
		// 	// One output file - one batch, dummy intervals
		// 	intervals := make([][]int64, 1)
		// 	intervals[0] = []int64{int64(0), 0}
		// 	return intervals, nil
	} else if node.HasFileReader() {
		// One input file - one batch
		intervals := make([][]int64, len(node.FileReader.SrcFileUrls))
		for i := 0; i < len(node.FileReader.SrcFileUrls); i++ {
			intervals[i] = []int64{int64(i), int64(i)}
		}
		return intervals, nil
	}

	return nil, fmt.Errorf("cannot find implementation for intervals for node %s", node.Name)
}

func (node *ScriptNodeDef) isNodeUsesIdx(idxName string) bool {
	if node.HasLookup() && node.Lookup.IndexName == idxName {
		return true
	}

	distinctIdxCandidate, ok := node.TableCreator.Indexes[idxName]
	if ok {
		if node.Type == NodeTypeDistinctTable && distinctIdxCandidate.Uniqueness == IdxUnique {
			return true
		}
	}

	return false
}
