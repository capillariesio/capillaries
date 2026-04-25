package sc

import (
	"encoding/json"
	"fmt"

	"github.com/capillariesio/capillaries/pkg/xfer"
)

type ScriptInitResult struct {
	Def         *ScriptDef
	InitProblem ScriptInitProblemType
	Err         error
}

func NewScriptFromFiles(caPath string, privateKeys map[string]string, scriptUrl string, scriptParamsUrl string, customProcessorDefFactoryInstance CustomProcessorDefFactory, customProcessorsSettings map[string]json.RawMessage) (*ScriptDef, ScriptInitProblemType, error) {

	scriptCacheKey := fmt.Sprintf("%s %s", scriptUrl, scriptParamsUrl)
	if ScriptDefCache != nil {
		cachedScriptInitResult, ok := ScriptDefCache.Get(scriptCacheKey)
		if ok {
			ScriptDefCacheHitCounter.Inc()
			return cachedScriptInitResult.Def, cachedScriptInitResult.InitProblem, cachedScriptInitResult.Err
		}
		ScriptDefCacheMissCounter.Inc()
	}

	jsonBytesScript, err := xfer.GetFileBytes(scriptUrl, caPath, privateKeys)
	if err != nil {
		return nil, ScriptInitConnectivityProblem, fmt.Errorf("cannot read script %s: %s", scriptUrl, err.Error())
	}

	var jsonBytesParams []byte
	if scriptParamsUrl != "" {
		jsonBytesParams, err = xfer.GetFileBytes(scriptParamsUrl, caPath, privateKeys)
		if err != nil {
			return nil, ScriptInitConnectivityProblem, fmt.Errorf("cannot read script parameters %s: %s", scriptParamsUrl, err.Error())
		}
	}

	scriptDef, initProblem, err := NewScriptFromFileBytes(caPath, privateKeys, scriptUrl, jsonBytesScript, scriptParamsUrl, jsonBytesParams, customProcessorDefFactoryInstance, customProcessorsSettings)
	if ScriptDefCache != nil && initProblem != ScriptInitConnectivityProblem {
		ScriptDefCache.Add(scriptCacheKey, ScriptInitResult{scriptDef, initProblem, err})
	}
	return scriptDef, initProblem, err
}
