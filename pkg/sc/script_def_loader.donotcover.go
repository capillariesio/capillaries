package sc

import (
	"encoding/json"
	"fmt"

	"github.com/capillariesio/capillaries/pkg/xfer"
	"github.com/hashicorp/golang-lru/v2/expirable"
)

func NewScriptFromFiles(scriptCache *expirable.LRU[string, string], caPath string, privateKeys map[string]string, scriptUri string, scriptParamsUri string, customProcessorDefFactoryInstance CustomProcessorDefFactory, customProcessorsSettings map[string]json.RawMessage) (*ScriptDef, ScriptInitProblemType, error) {
	var err error
	var jsonBytesScript []byte
	var jsonBytesParams []byte

	if scriptCache != nil {
		if jsonCachedScript, ok := scriptCache.Get(scriptUri); ok {
			jsonBytesScript = []byte(jsonCachedScript)
		}
		if scriptParamsUri != "" {
			if jsonCachedParams, ok := scriptCache.Get(scriptParamsUri); ok {
				jsonBytesParams = []byte(jsonCachedParams)
			}
		}
	}

	if jsonBytesScript == nil {
		jsonBytesScript, err = xfer.GetFileBytes(scriptUri, caPath, privateKeys)
		if err != nil {
			return nil, ScriptInitConnectivityProblem, fmt.Errorf("cannot read script %s: %s", scriptUri, err.Error())
		}
		if scriptCache != nil {
			scriptCache.Add(scriptUri, string(jsonBytesScript))
		}
	}

	if jsonBytesParams == nil && scriptParamsUri != "" {
		jsonBytesParams, err = xfer.GetFileBytes(scriptParamsUri, caPath, privateKeys)
		if err != nil {
			return nil, ScriptInitConnectivityProblem, fmt.Errorf("cannot read script parameters %s: %s", scriptParamsUri, err.Error())
		}
		if scriptCache != nil {
			scriptCache.Add(scriptParamsUri, string(jsonBytesParams))
		}
	}

	return NewScriptFromFileBytes(caPath, privateKeys, scriptUri, jsonBytesScript, scriptParamsUri, jsonBytesParams, customProcessorDefFactoryInstance, customProcessorsSettings)
}
