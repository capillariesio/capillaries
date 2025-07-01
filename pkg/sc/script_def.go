package sc

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const RunIdSuffixLen int = 6             // _00001
const RunIdSuffixFormat string = "_%05d" // _00001
const MaxTableNameLen int = 42           // Amazon keyspaces 48, minus RunIdSuffixLen

const (
	ReservedParamBatchIdx string = "{batch_idx|string}"
	ReservedParamRunId    string = "{run_id|string}"
)

type ScriptType string

const (
	ScriptJson    ScriptType = "json"
	ScriptYaml    ScriptType = "yaml"
	ScriptUnknown ScriptType = "unknown"
)

type ScriptDef struct {
	ScriptNodes           map[string]*ScriptNodeDef  `json:"nodes" yaml:"nodes"`
	RawDependencyPolicies map[string]json.RawMessage `json:"dependency_policies" yaml:"dependency_policies"`
	TableCreatorNodeMap   map[string](*ScriptNodeDef)
	IndexNodeMap          map[string](*ScriptNodeDef)
}

func (scriptDef *ScriptDef) buildIndexNodeMap() error {
	scriptDef.IndexNodeMap = map[string]*ScriptNodeDef{}
	for _, node := range scriptDef.ScriptNodes {
		if node.HasTableCreator() {
			for idxName := range node.TableCreator.Indexes {
				if _, ok := scriptDef.IndexNodeMap[idxName]; ok {
					return fmt.Errorf("duplicate index name: %s", idxName)
				}
				if _, ok := scriptDef.TableCreatorNodeMap[idxName]; ok {
					return fmt.Errorf("cannot use same name for table and index: %s", idxName)
				}
				scriptDef.IndexNodeMap[idxName] = node
			}
		}
	}
	return nil
}

func (scriptDef *ScriptDef) buildTableCreatorNodeMap() error {
	scriptDef.TableCreatorNodeMap = map[string]*ScriptNodeDef{}
	for _, node := range scriptDef.ScriptNodes {
		if node.HasTableCreator() {
			if _, ok := scriptDef.TableCreatorNodeMap[node.TableCreator.Name]; ok {
				return fmt.Errorf("duplicate table name: %s", node.TableCreator.Name)
			}
			scriptDef.TableCreatorNodeMap[node.TableCreator.Name] = node
		}
	}
	return nil
}

func (scriptDef *ScriptDef) checkDependencyPolicyUsage(scriptType ScriptType) error {
	depPolMap := map[string](*DependencyPolicyDef){}
	defaultDepPolCount := 0
	var defaultDepPol *DependencyPolicyDef
	for polName, rawPolDef := range scriptDef.RawDependencyPolicies {
		pol := DependencyPolicyDef{}
		if err := pol.Deserialize(rawPolDef, scriptType); err != nil {
			return fmt.Errorf("failed to deserialize dependency policy %s: %s", polName, err.Error())
		}
		depPolMap[polName] = &pol
		if pol.IsDefault {
			defaultDepPol = &pol
			defaultDepPolCount++
		}
	}
	if defaultDepPolCount != 1 {
		return fmt.Errorf("failed to deserialize dependency policies, found %d default policies, required 1", defaultDepPolCount)
	}

	for polName, polDef := range depPolMap {
		if err := polDef.evalRuleExpressionsAndCheckType(); err != nil {
			return fmt.Errorf("failed to test dependency policy %s rules: %s", polName, err.Error())
		}
	}

	for _, node := range scriptDef.ScriptNodes {
		if node.HasTableReader() {
			if len(node.DependencyPolicyName) == 0 {
				node.DepPolDef = defaultDepPol
			} else {
				var ok bool
				node.DepPolDef, ok = depPolMap[node.DependencyPolicyName]
				if !ok {
					return fmt.Errorf("cannot find dependency policy %s for node %s", node.DependencyPolicyName, node.Name)
				}
			}
		}
	}
	return nil
}

func (scriptDef *ScriptDef) Deserialize(jsonOrYamlBytesScript []byte, scriptType ScriptType, customProcessorDefFactory CustomProcessorDefFactory, customProcessorsSettings map[string]json.RawMessage, caPath string, privateKeys map[string]string) error {

	if err := JsonOrYamlUnmarshal(scriptType, jsonOrYamlBytesScript, &scriptDef); err != nil {
		return fmt.Errorf("cannot unmarshal script: [%s]", err.Error())
	}

	foundErrors := make([]string, 0, 2)

	// Deserialize node by node
	for nodeName, node := range scriptDef.ScriptNodes {
		node.Name = nodeName
		if err := node.Deserialize(customProcessorDefFactory, customProcessorsSettings, scriptType, caPath, privateKeys); err != nil {
			foundErrors = append(foundErrors, fmt.Sprintf("cannot deserialize node %s: [%s]", nodeName, err.Error()))
		}
	}

	if len(foundErrors) > 0 {
		return fmt.Errorf("%s", strings.Join(foundErrors, "; "))
	}

	// Table -> node map, to look for ord and lkp indexes, for those nodes that create tables
	if err := scriptDef.buildTableCreatorNodeMap(); err != nil {
		return err
	}

	// Index -> node map, to look for ord and lkp indexes, for those nodes that create tables
	if err := scriptDef.buildIndexNodeMap(); err != nil {
		return err
	}

	for _, node := range scriptDef.ScriptNodes {
		if err := scriptDef.resolveReader(node); err != nil {
			return fmt.Errorf("failed to resolve reader for node %s: [%s]", node.Name, err.Error())
		}
	}

	for _, node := range scriptDef.ScriptNodes {
		if err := scriptDef.resolveLookup(node); err != nil {
			return fmt.Errorf("failed to resolve lookup for node %s: [%s]", node.Name, err.Error())
		}
	}

	for idxName, creatorNodeDef := range scriptDef.IndexNodeMap {
		if !scriptDef.isScriptUsesIdx(idxName) {
			// TODO: this is a hack to allow indexes that are deliberately added to check uniqueness without Capillaries complaining "this idx is not used"
			if !strings.Contains(idxName, "to_check_uniq") {
				return fmt.Errorf("cannot find nodes that use index %s created by node %s, consider removing this index", idxName, creatorNodeDef.Name)
			}
		}
	}

	for _, node := range scriptDef.ScriptNodes {
		if err := scriptDef.checkFieldUsageInCustomProcessorCreator(node); err != nil {
			return fmt.Errorf("field usage error in custom processor creator (check custom artifacts, like criteria files), node %s: [%s]", node.Name, err.Error())
		}
	}

	for _, node := range scriptDef.ScriptNodes {
		if err := scriptDef.checkFieldUsageInCreator(node); err != nil {
			return fmt.Errorf("field usage error in creator, node %s: [%s]", node.Name, err.Error())
		}
	}

	for _, node := range scriptDef.ScriptNodes {
		if err := node.evalCreatorAndLookupExpressionsAndCheckType(); err != nil {
			return fmt.Errorf("failed evaluating creator/lookup expressions for node %s: [%s]", node.Name, err.Error())
		}
	}

	return scriptDef.checkDependencyPolicyUsage(scriptType)
}

func (scriptDef *ScriptDef) resolveReader(node *ScriptNodeDef) error {
	if node.HasTableReader() {
		tableCreatorNode, ok := scriptDef.TableCreatorNodeMap[node.TableReader.TableName]
		if !ok {
			return fmt.Errorf("cannot find the node that creates table [%s]", node.TableReader.TableName)
		}
		node.TableReader.TableCreator = &tableCreatorNode.TableCreator
	}
	return nil
}

func (scriptDef *ScriptDef) resolveLookup(node *ScriptNodeDef) error {
	if !node.HasLookup() {
		return nil
	}

	srcFieldRefs, err := node.getSourceFieldRefs()
	if err != nil {
		return fmt.Errorf("unexpectedly cannot resolve source field refs: [%s]", err.Error())
	}
	idxCreatorNode, ok := scriptDef.IndexNodeMap[node.Lookup.IndexName]
	if !ok {
		return fmt.Errorf("cannot find the node that creates index [%s]", node.Lookup.IndexName)
	}

	node.Lookup.TableCreator = &idxCreatorNode.TableCreator

	if err = node.Lookup.resolveLeftTableFields(ReaderAlias, srcFieldRefs); err != nil {
		return err
	}

	if err = node.Lookup.ParseFilter(); err != nil {
		return err
	}

	if err = node.Lookup.ValidateJoinType(); err != nil {
		return err
	}

	return node.Lookup.CheckPagedBatchSize()
}

func (scriptDef *ScriptDef) checkFieldUsageInCreator(node *ScriptNodeDef) error {
	srcFieldRefs, err := node.getSourceFieldRefs()
	if err != nil {
		return fmt.Errorf("unexpectedly cannot resolve source field refs: [%s]", err.Error())
	}

	var processorFieldRefs *FieldRefs
	if node.HasCustomProcessor() {
		processorFieldRefs = node.CustomProcessor.GetFieldRefs()
		if err != nil {
			return fmt.Errorf("cannot resolve processor field refs: [%s]", err.Error())
		}
	}

	var lookupFieldRefs *FieldRefs
	if node.HasLookup() {
		lookupFieldRefs = node.Lookup.TableCreator.GetFieldRefsWithAlias(LookupAlias)
	}

	foundErrors := make([]string, 0)

	var targetFieldRefs *FieldRefs
	if node.HasTableCreator() {
		targetFieldRefs = node.TableCreator.GetFieldRefsWithAlias(CreatorAlias)
	} else if node.HasFileCreator() {
		targetFieldRefs = node.FileCreator.getFieldRefs()
	} else {
		return errors.New("dev error, unknown creator")
	}

	// Lookup
	if node.HasLookup() && node.Lookup.UsesFilter() {
		// Having: allow only lookup table, prohibit src and tgt
		if err := checkAllowed(&node.Lookup.UsedInFilterFields, JoinFieldRefs(srcFieldRefs, targetFieldRefs), lookupFieldRefs); err != nil {
			foundErrors = append(foundErrors, fmt.Sprintf("invalid field in lookup filter [%s], only fields from the lookup table [%s](alias %s) are allowed: [%s]", node.Lookup.RawFilter, node.Lookup.TableCreator.Name, LookupAlias, err.Error()))
		}
	}

	// Table creator
	if node.HasTableCreator() {
		srcLkpCustomFieldRefs := JoinFieldRefs(srcFieldRefs, lookupFieldRefs, processorFieldRefs)
		// Having: allow tgt fields, prohibit src, lkp
		if err := checkAllowed(&node.TableCreator.UsedInHavingFields, srcLkpCustomFieldRefs, targetFieldRefs); err != nil {
			foundErrors = append(foundErrors, fmt.Sprintf("invalid field in table creator 'having' condition: [%s]; only target (w.*) fields allowed, reader (r.*) and lookup (l.*) fields are prohibited", err.Error()))
		}
		// Tgt expressions: allow src iterator table (or src file), lkp, custom processor, prohibit target
		// TODO: aggregate functions cannot include fields from group field list
		if err := checkAllowed(&node.TableCreator.UsedInTargetExpressionsFields, targetFieldRefs, srcLkpCustomFieldRefs); err != nil {
			foundErrors = append(foundErrors, fmt.Sprintf("invalid field(s) in target table field expression: [%s]", err.Error()))
		}
	}

	// File creator
	if node.HasFileCreator() {
		// Having: allow tgt fields, prohibit src
		if err := checkAllowed(&node.FileCreator.UsedInHavingFields, srcFieldRefs, targetFieldRefs); err != nil {
			foundErrors = append(foundErrors, fmt.Sprintf("invalid field in file creator 'having' condition: [%s]", err.Error()))
		}

		// Tgt expressions: allow src, prohibit target fields
		// TODO: aggregate functions cannot include fields from group field list
		if err := checkAllowed(&node.FileCreator.UsedInTargetExpressionsFields, targetFieldRefs, srcFieldRefs); err != nil {
			foundErrors = append(foundErrors, fmt.Sprintf("invalid field in target file field expression: [%s]", err.Error()))
		}
	}

	if len(foundErrors) > 0 {
		return fmt.Errorf("%s", strings.Join(foundErrors, "; "))
	}

	return nil
}

func (scriptDef *ScriptDef) checkFieldUsageInCustomProcessorCreator(node *ScriptNodeDef) error {
	if !node.HasCustomProcessor() {
		return nil
	}

	srcFieldRefs, err := node.getSourceFieldRefs()
	if err != nil {
		return fmt.Errorf("unexpectedly cannot resolve source field refs: [%s]", err.Error())
	}

	procTgtFieldRefs := node.CustomProcessor.GetFieldRefs()

	// In processor fields, we are allowed to use only reader and processor fields ("r" and "p")
	if err := checkAllowed(node.CustomProcessor.GetUsedInTargetExpressionsFields(), nil, JoinFieldRefs(srcFieldRefs, procTgtFieldRefs)); err != nil {
		return fmt.Errorf("invalid field(s) in target table field expression: [%s]", err.Error())
	}

	return nil
}

func (scriptDef *ScriptDef) addChildrenToManual(rootNode *ScriptNodeDef, manualSet map[string]struct{}, startSet map[string]struct{}) {
	_, isRootInManual := manualSet[rootNode.Name]
	_, isRootInStart := startSet[rootNode.Name]
	for _, node := range scriptDef.ScriptNodes {
		if rootNode.HasTableCreator() && node.HasTableReader() && rootNode.TableCreator.Name == node.TableReader.TableName && (isRootInManual && !isRootInStart || node.StartPolicy == NodeStartManual) {
			manualSet[node.Name] = struct{}{}
			scriptDef.addChildrenToManual(node, manualSet, startSet)
		} else if rootNode.HasTableCreator() && node.HasLookup() && rootNode.TableCreator.Name == node.Lookup.TableCreator.Name && (isRootInManual && !isRootInStart || node.StartPolicy == NodeStartManual) {
			manualSet[node.Name] = struct{}{}
			scriptDef.addChildrenToManual(node, manualSet, startSet)
		}
	}
}

func (scriptDef *ScriptDef) addChildrenToAffected(rootNode *ScriptNodeDef, affectedSet map[string]struct{}, manualSet map[string]struct{}) {
	for _, node := range scriptDef.ScriptNodes {
		_, isCurrentInManual := manualSet[node.Name]
		if rootNode.HasTableCreator() && node.HasTableReader() && rootNode.TableCreator.Name == node.TableReader.TableName && !isCurrentInManual {
			affectedSet[node.Name] = struct{}{}
			scriptDef.addChildrenToAffected(node, affectedSet, manualSet)
		} else if rootNode.HasTableCreator() && node.HasLookup() && rootNode.TableCreator.Name == node.Lookup.TableCreator.Name && !isCurrentInManual {
			affectedSet[node.Name] = struct{}{}
			scriptDef.addChildrenToAffected(node, affectedSet, manualSet)
		}
	}
}

// Returns all nodes that will receive RabbitMQ messages when a run is started with startNodeNames
// The tricky part is not to include nodes that have "manual" nodes between them and the start nodes (see addChildrenToManual)
func (scriptDef *ScriptDef) GetAffectedNodes(startNodeNames []string) []string {
	startSet := map[string]struct{}{}
	for _, nodeName := range startNodeNames {
		startSet[nodeName] = struct{}{}
	}
	manualSet := map[string]struct{}{}
	for _, nodeName := range startNodeNames {
		if node, ok := scriptDef.ScriptNodes[nodeName]; ok {
			scriptDef.addChildrenToManual(node, manualSet, startSet)
		}
	}

	affectedSet := map[string]struct{}{}
	for _, nodeName := range startNodeNames {
		if node, ok := scriptDef.ScriptNodes[nodeName]; ok {
			affectedSet[node.Name] = struct{}{}
			scriptDef.addChildrenToAffected(node, affectedSet, manualSet)
		}
	}

	affectedList := make([]string, len(affectedSet))
	i := 0
	for k := range affectedSet {
		affectedList[i] = k
		i++
	}
	return affectedList
}

func (scriptDef *ScriptDef) isScriptUsesIdx(idxName string) bool {
	for _, node := range scriptDef.ScriptNodes {
		if node.isNodeUsesIdx(idxName) {
			return true
		}
	}
	return false
}
