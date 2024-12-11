package sc

import (
	"encoding/json"
	"fmt"

	"github.com/capillariesio/capillaries/pkg/xfer"
	"github.com/hashicorp/golang-lru/v2/expirable"
)

func NewScriptFromFiles(scriptCache *expirable.LRU[string, string], caPath string, privateKeys map[string]string, scriptUrl string, scriptParamsUrl string, customProcessorDefFactoryInstance CustomProcessorDefFactory, customProcessorsSettings map[string]json.RawMessage) (*ScriptDef, ScriptInitProblemType, error) {
	var err error
	var jsonBytesScript []byte
	var jsonBytesParams []byte

	if scriptCache != nil {
		if jsonCachedScript, ok := scriptCache.Get(scriptUrl); ok {
			jsonBytesScript = []byte(jsonCachedScript)
		}
		if scriptParamsUrl != "" {
			if jsonCachedParams, ok := scriptCache.Get(scriptParamsUrl); ok {
				jsonBytesParams = []byte(jsonCachedParams)
			}
		}
	}

	if jsonBytesScript == nil {
		jsonBytesScript, err = xfer.GetFileBytes(scriptUrl, caPath, privateKeys)
		if err != nil {
			return nil, ScriptInitConnectivityProblem, fmt.Errorf("cannot read script %s: %s", scriptUrl, err.Error())
		}
		if scriptCache != nil {
			scriptCache.Add(scriptUrl, string(jsonBytesScript))
		}
	}

	if jsonBytesParams == nil && scriptParamsUrl != "" {
		jsonBytesParams, err = xfer.GetFileBytes(scriptParamsUrl, caPath, privateKeys)
		if err != nil {
			return nil, ScriptInitConnectivityProblem, fmt.Errorf("cannot read script parameters %s: %s", scriptParamsUrl, err.Error())
		}
		if scriptCache != nil {
			scriptCache.Add(scriptParamsUrl, string(jsonBytesParams))
		}
	}

	return NewScriptFromFileBytes(caPath, privateKeys, scriptUrl, jsonBytesScript, scriptParamsUrl, jsonBytesParams, customProcessorDefFactoryInstance, customProcessorsSettings)
}
