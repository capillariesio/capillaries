package sc

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/capillariesio/capillaries/pkg/xfer"
)

const (
	ReservedParamBatchIdx string = "{batch_idx|string}"
	ReservedParamRunId    string = "{run_id|string}"
)

type ScriptDef struct {
	ScriptNodes           map[string]*ScriptNodeDef  `json:"nodes"`
	RawDependencyPolicies map[string]json.RawMessage `json:"dependency_policies"`
	TableCreatorNodeMap   map[string](*ScriptNodeDef)
	IndexNodeMap          map[string](*ScriptNodeDef)
}

func (scriptDef *ScriptDef) Deserialize(jsonBytesScript []byte, customProcessorDefFactory CustomProcessorDefFactory, customProcessorsSettings map[string]json.RawMessage, caPath string, privateKeys map[string]string) error {

	if err := json.Unmarshal(jsonBytesScript, &scriptDef); err != nil {
		return fmt.Errorf("cannot unmarshal script json: [%s]", err.Error())
	}

	errors := make([]string, 0, 2)

	// Deserialize node by node
	for nodeName, node := range scriptDef.ScriptNodes {
		node.Name = nodeName
		if err := node.Deserialize(customProcessorDefFactory, customProcessorsSettings, caPath, privateKeys); err != nil {
			errors = append(errors, fmt.Sprintf("cannot deserialize node %s: [%s]", nodeName, err.Error()))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, "; "))
	}

	// Table -> node map, to look for ord and lkp indexes, for those nodes that create tables
	scriptDef.TableCreatorNodeMap = map[string]*ScriptNodeDef{}
	for _, node := range scriptDef.ScriptNodes {
		if node.HasTableCreator() {
			if _, ok := scriptDef.TableCreatorNodeMap[node.TableCreator.Name]; ok {
				return fmt.Errorf("duplicate table name: %s", node.TableCreator.Name)
			}
			scriptDef.TableCreatorNodeMap[node.TableCreator.Name] = node
		}
	}
	// Index -> node map, to look for ord and lkp indexes, for those nodes that create tables
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

	for _, node := range scriptDef.ScriptNodes {
		if err := scriptDef.checkFieldUsageInCustomProcessor(node); err != nil {
			return fmt.Errorf("field usage error in custom processor, node %s: [%s]", node.Name, err.Error())
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

	depPolMap := map[string](*DependencyPolicyDef){}
	defaultDepPolCount := 0
	var defaultDepPol *DependencyPolicyDef
	for polName, rawPolDef := range scriptDef.RawDependencyPolicies {
		pol := DependencyPolicyDef{}
		if err := pol.Deserialize(rawPolDef); err != nil {
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

func NewScriptFromFiles(caPath string, privateKeys map[string]string, scriptUri string, scriptParamsUri string, customProcessorDefFactoryInstance CustomProcessorDefFactory, customProcessorsSettings map[string]json.RawMessage) (*ScriptDef, error) {
	jsonBytesScript, err := xfer.GetFileBytes(scriptUri, caPath, privateKeys)
	if err != nil {
		return nil, fmt.Errorf("cannot read script: %s", err.Error())
	}

	// Make sure parameters are in canonical format: {param_name|param_type}
	scriptString := string(jsonBytesScript)

	// Default param type is string: {param} -> {param|string}
	re := regexp.MustCompile("{[ ]*([a-zA-Z0-9_]+)[ ]*}")
	scriptString = re.ReplaceAllString(scriptString, "{$1|string}")

	// Remove spaces: {  param_name | param_type } -> {param_name|param_type}
	re = regexp.MustCompile(`{[ ]*([a-zA-Z0-9_]+)[ ]*\|[ ]*(string|number|bool)[ ]*}`)
	scriptString = re.ReplaceAllString(scriptString, "{$1|$2}")

	// Verify that number/bool must be like "{param_name|number}", no extra characters between double quotes and curly braces
	re = regexp.MustCompile(`([^"]{[a-zA-Z0-9_]+\|(number|bool)})|({[a-zA-Z0-9_]+\|(number|bool)}[^"])`)
	invalidParamRefs := re.FindAllString(scriptString, -1)
	if len(invalidParamRefs) > 0 {
		return nil, fmt.Errorf("cannot parse number/bool script parameter references in [%s], the following parameter references should not have extra characters between curly braces and double quotes: [%s]", scriptUri, strings.Join(invalidParamRefs, ","))
	}

	var jsonBytesParams []byte
	if len(scriptParamsUri) > 0 {
		jsonBytesParams, err = xfer.GetFileBytes(scriptParamsUri, caPath, privateKeys)
		if err != nil {
			return nil, fmt.Errorf("cannot read script parameters: %s", err.Error())
		}
	}

	// Apply template params here, script def should know nothing about them: they may tweak some 3d-party tfm config

	paramsMap := map[string]interface{}{}
	if jsonBytesParams != nil {
		if err := json.Unmarshal(jsonBytesParams, &paramsMap); err != nil {
			return nil, fmt.Errorf("cannot unmarshal script params json from [%s]: [%s]", scriptParamsUri, err.Error())
		}
	}

	replacerStrings := make([]string, len(paramsMap)*2)
	i := 0
	for templateParam, templateParamVal := range paramsMap {
		switch typedParamVal := templateParamVal.(type) {
		case string:
			// Revert \n unescaping in parameter values - we want to preserve "\n"
			if strings.Contains(typedParamVal, "\n") {
				typedParamVal = strings.ReplaceAll(typedParamVal, "\n", "\\n")
			}
			// Just replace {param_name|string} with value, pay no attention to double quotes
			replacerStrings[i] = fmt.Sprintf("{%s|string}", templateParam)
			replacerStrings[i+1] = typedParamVal
		case float64:
			// Expect enclosed in double quotes
			replacerStrings[i] = fmt.Sprintf(`"{%s|number}"`, templateParam)
			if typedParamVal == float64(int64(typedParamVal)) {
				replacerStrings[i+1] = fmt.Sprintf("%d", int64(typedParamVal))
			} else {
				replacerStrings[i+1] = fmt.Sprintf("%f", typedParamVal)
			}
		case bool:
			// Expect enclosed in double quotes
			replacerStrings[i] = fmt.Sprintf(`"{%s|bool}"`, templateParam)
			replacerStrings[i+1] = fmt.Sprintf("%t", typedParamVal)
		default:
			return nil, fmt.Errorf("unsupported parameter type %T from [%s]: %s", templateParamVal, scriptParamsUri, templateParam)
		}
		i += 2
	}
	scriptString = strings.NewReplacer(replacerStrings...).Replace(scriptString)

	// Verify all parameters were replaced
	re = regexp.MustCompile(`({[a-zA-Z0-9_]+\|(string|number|bool)})`)
	unresolvedParamRefs := re.FindAllString(scriptString, -1)
	unresolvedParamMap := map[string]struct{}{}
	reservedParamRefs := map[string]struct{}{ReservedParamBatchIdx: {}}
	for _, paramRef := range unresolvedParamRefs {
		if _, ok := reservedParamRefs[paramRef]; !ok {
			unresolvedParamMap[paramRef] = struct{}{}
		}
	}
	if len(unresolvedParamMap) > 0 {
		return nil, fmt.Errorf("unresolved parameter references in [%s]: %v; make sure that type in the script matches the type of the parameter value in the script parameters file", scriptUri, unresolvedParamMap)
	}

	newScript := &ScriptDef{}
	if err = newScript.Deserialize([]byte(scriptString), customProcessorDefFactoryInstance, customProcessorsSettings, caPath, privateKeys); err != nil {
		return nil, fmt.Errorf("cannot deserialize script %s(%s): %s", scriptUri, scriptParamsUri, err.Error())
	}

	return newScript, nil
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
		return fmt.Errorf("cannot resolve source field refs: [%s]", err.Error())
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

	if err = node.Lookup.CheckPagedBatchSize(); err != nil {
		return err
	}

	return nil

}

func (scriptDef *ScriptDef) checkFieldUsageInCreator(node *ScriptNodeDef) error {
	srcFieldRefs, err := node.getSourceFieldRefs()
	if err != nil {
		return fmt.Errorf("cannot resolve source field refs: [%s]", err.Error())
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

	errors := make([]string, 0)

	var targetFieldRefs *FieldRefs
	if node.HasTableCreator() {
		targetFieldRefs = node.TableCreator.GetFieldRefsWithAlias(CreatorAlias)
	} else if node.HasFileCreator() {
		targetFieldRefs = node.FileCreator.getFieldRefs()
	} else {
		return fmt.Errorf("dev error, unknown creator")
	}

	// Lookup
	if node.HasLookup() && node.Lookup.UsesFilter() {
		// Having: allow only lookup table, prohibit src and tgt
		if err := checkAllowed(&node.Lookup.UsedInFilterFields, JoinFieldRefs(srcFieldRefs, targetFieldRefs), lookupFieldRefs); err != nil {
			errors = append(errors, fmt.Sprintf("invalid field in lookup filter [%s], only fields from the lookup table [%s](alias %s) are allowed: [%s]", node.Lookup.RawFilter, node.Lookup.TableCreator.Name, LookupAlias, err.Error()))
		}
	}

	// Table creator
	if node.HasTableCreator() {
		srcLkpCustomFieldRefs := JoinFieldRefs(srcFieldRefs, lookupFieldRefs, processorFieldRefs)
		// Having: allow tgt fields, prohibit src, lkp
		if err := checkAllowed(&node.TableCreator.UsedInHavingFields, srcLkpCustomFieldRefs, targetFieldRefs); err != nil {
			errors = append(errors, fmt.Sprintf("invalid field in table creator 'having' condition: [%s]", err.Error()))
		}
		// Tgt expressions: allow src iterator table (or src file), lkp, custom processor, prohibit target
		// TODO: aggregate functions cannot include fields from group field list
		if err := checkAllowed(&node.TableCreator.UsedInTargetExpressionsFields, targetFieldRefs, srcLkpCustomFieldRefs); err != nil {
			errors = append(errors, fmt.Sprintf("invalid field(s) in target table field expression: [%s]", err.Error()))
		}
	}

	// File creator
	if node.HasFileCreator() {
		// Having: allow tgt fields, prohibit src
		if err := checkAllowed(&node.FileCreator.UsedInHavingFields, srcFieldRefs, targetFieldRefs); err != nil {
			errors = append(errors, fmt.Sprintf("invalid field in file creator 'having' condition: [%s]", err.Error()))
		}

		// Tgt expressions: allow src, prohibit target fields
		// TODO: aggregate functions cannot include fields from group field list
		if err := checkAllowed(&node.FileCreator.UsedInTargetExpressionsFields, targetFieldRefs, srcFieldRefs); err != nil {
			errors = append(errors, fmt.Sprintf("invalid field in target file field expression: [%s]", err.Error()))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, "; "))
	} else {
		return nil
	}
}

func (scriptDef *ScriptDef) checkFieldUsageInCustomProcessor(node *ScriptNodeDef) error {
	if !node.HasCustomProcessor() {
		return nil
	}

	srcFieldRefs, err := node.getSourceFieldRefs()
	if err != nil {
		return fmt.Errorf("cannot resolve source field refs: [%s]", err.Error())
	}

	procTgtFieldRefs := node.CustomProcessor.GetFieldRefs()

	// In processor fields, we are allowed to use only reader and processor fields ("r" and "p")
	if err := checkAllowed(node.CustomProcessor.GetUsedInTargetExpressionsFields(), nil, JoinFieldRefs(srcFieldRefs, procTgtFieldRefs)); err != nil {
		return fmt.Errorf("invalid field(s) in target table field expression: [%s]", err.Error())
	}

	return nil
}

// func (scriptDef *ScriptDef) HarvestNodesReadingFromThis(rootNode *ScriptNodeDef) []*ScriptNodeDef {
// 	dependentNodes := make([]*ScriptNodeDef, len(scriptDef.ScriptNodes))
// 	dependentNodeCount := 0
// 	for _, node := range scriptDef.ScriptNodes {
// 		if node.HasTableReader() && rootNode.TableCreator.Name == node.TableReader.TableName {
// 			dependentNodes[dependentNodeCount] = node
// 			dependentNodeCount++
// 		}
// 	}
// 	return dependentNodes[:dependentNodeCount]
// }

func (scriptDef *ScriptDef) addToAffected(rootNode *ScriptNodeDef, affectedSet map[string]struct{}) {
	if _, ok := affectedSet[rootNode.Name]; ok {
		return
	}

	affectedSet[rootNode.Name] = struct{}{}

	for _, node := range scriptDef.ScriptNodes {
		if rootNode.HasTableCreator() && node.HasTableReader() && rootNode.TableCreator.Name == node.TableReader.TableName && node.StartPolicy == NodeStartAuto {
			scriptDef.addToAffected(node, affectedSet)
		} else if rootNode.HasTableCreator() && node.HasLookup() && rootNode.TableCreator.Name == node.Lookup.TableCreator.Name && node.StartPolicy == NodeStartAuto {
			scriptDef.addToAffected(node, affectedSet)
		}
	}
}

func (scriptDef *ScriptDef) GetAffectedNodes(startNodeNames []string) []string {
	affectedSet := map[string]struct{}{}
	for _, nodeName := range startNodeNames {
		if node, ok := scriptDef.ScriptNodes[nodeName]; ok {
			scriptDef.addToAffected(node, affectedSet)
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
