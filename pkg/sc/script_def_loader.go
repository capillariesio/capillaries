package sc

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

type ScriptInitProblemType int

const ScriptInitNoProblem ScriptInitProblemType = 0
const ScriptInitUrlProblem ScriptInitProblemType = 1
const ScriptInitContentProblem ScriptInitProblemType = 2
const ScriptInitConnectivityProblem ScriptInitProblemType = 3

func NewScriptFromFileBytes(caPath string, privateKeys map[string]string, scriptUri string, jsonBytesScript []byte, scriptParamsUri string, jsonOrYamlBytesParams []byte, customProcessorDefFactoryInstance CustomProcessorDefFactory, customProcessorsSettings map[string]json.RawMessage) (*ScriptDef, ScriptInitProblemType, error) {
	// Make sure parameters are in canonical format: {param_name|param_type}
	scriptString := string(jsonBytesScript)

	// Default param type is string: {param} -> {param|string}
	re := regexp.MustCompile("{[ ]*([a-zA-Z0-9_]+)[ ]*}")
	scriptString = re.ReplaceAllString(scriptString, "{$1|string}")

	// Remove spaces: {  param_name | param_type } -> {param_name|param_type}
	re = regexp.MustCompile(`{[ ]*([a-zA-Z0-9_]+)[ ]*\|[ ]*(string|number|bool|stringlist)[ ]*}`)
	scriptString = re.ReplaceAllString(scriptString, "{$1|$2}")

	// Verify that number/bool must be like "{param_name|number}", no extra characters between double quotes and curly braces
	// This is a big limitation of the existing parameter implementation.
	// Those double quetes must be there in order to keep JSON well-formed.
	// Because of this, using number/bool parameters in strings is not allowed, use string paramater like "10" and "false" in those cases.
	re = regexp.MustCompile(`([^"]{[a-zA-Z0-9_]+\|(number|bool)})|({[a-zA-Z0-9_]+\|(number|bool)}[^"])`)
	invalidParamRefs := re.FindAllString(scriptString, -1)
	if len(invalidParamRefs) > 0 {
		return nil, ScriptInitUrlProblem, fmt.Errorf("cannot parse number/bool script parameter references in [%s], the following parameter references should not have extra characters between curly braces and double quotes: [%s]", scriptUri, strings.Join(invalidParamRefs, ","))
	}

	// Apply template params here, script def should know nothing about them: they may tweak some 3d-party tfm config

	paramsMap := map[string]any{}
	if jsonOrYamlBytesParams != nil {
		scriptParamsType := ScriptUnknown
		if strings.HasSuffix(scriptParamsUri, ".json") {
			scriptParamsType = ScriptJson
		} else if strings.HasSuffix(scriptParamsUri, ".yaml") {
			scriptParamsType = ScriptYaml
		} else {
			return nil, ScriptInitContentProblem, fmt.Errorf("cannot unmarshal script params from [%s]: json or yaml extension expected", scriptParamsUri)
		}
		if err := JsonOrYamlUnmarshal(scriptParamsType, jsonOrYamlBytesParams, &paramsMap); err != nil {
			return nil, ScriptInitContentProblem, fmt.Errorf("cannot unmarshal script params from [%s]: [%s]", scriptParamsUri, err.Error())
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
				// It's an int in JSON
				replacerStrings[i+1] = fmt.Sprintf("%d", int64(typedParamVal))
			} else {
				// It's a float in JSON
				replacerStrings[i+1] = fmt.Sprintf("%f", typedParamVal)
			}
		case bool:
			// Expect enclosed in double quotes
			replacerStrings[i] = fmt.Sprintf(`"{%s|bool}"`, templateParam)
			replacerStrings[i+1] = fmt.Sprintf("%t", typedParamVal)
		default:
			arrayParamVal, ok := templateParamVal.([]any)
			if ok {
				switch arrayParamVal[0].(type) {
				case string:
					// It's a stringlist
					replacerStrings[i] = fmt.Sprintf(`"{%s|stringlist}"`, templateParam)
					strArray := make([]string, len(arrayParamVal))
					for i, itemAny := range arrayParamVal {
						itemStr, ok := itemAny.(string)
						if !ok {
							return nil, ScriptInitContentProblem, fmt.Errorf("stringlist contains non-string value type %T from [%s]: %s", itemAny, scriptParamsUri, templateParam)
						}
						strArray[i] = fmt.Sprintf(`"%s"`, itemStr)
					}
					replacerStrings[i+1] = fmt.Sprintf("[%s]", strings.Join(strArray, ","))
				default:
					return nil, ScriptInitContentProblem, fmt.Errorf("unsupported array parameter type %T from [%s]: %s", arrayParamVal, scriptParamsUri, templateParam)
				}
			} else {
				return nil, ScriptInitContentProblem, fmt.Errorf("unsupported parameter type %T from [%s]: %s", templateParamVal, scriptParamsUri, templateParam)
			}
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
		return nil, ScriptInitContentProblem, fmt.Errorf("unresolved parameter references in [%s]: %v; make sure that type in the script matches the type of the parameter value in the script parameters file", scriptUri, unresolvedParamMap)
	}

	newScript := &ScriptDef{}
	scriptType := ScriptUnknown
	if strings.HasSuffix(scriptUri, ".json") {
		scriptType = ScriptJson
	} else if strings.HasSuffix(scriptUri, ".yaml") {
		scriptType = ScriptYaml
	} else {
		return nil, ScriptInitContentProblem, fmt.Errorf("cannot unmarshal script from [%s]: json or yaml extension expected", scriptUri)
	}

	if err := newScript.Deserialize([]byte(scriptString), scriptType, customProcessorDefFactoryInstance, customProcessorsSettings, caPath, privateKeys); err != nil {
		return nil, ScriptInitContentProblem, fmt.Errorf("cannot deserialize script %s(%s): %s", scriptUri, scriptParamsUri, err.Error())
	}

	return newScript, ScriptInitNoProblem, nil
}
