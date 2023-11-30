package sc

import (
	"encoding/json"
	"fmt"

	"github.com/capillariesio/capillaries/pkg/xfer"
)

func NewScriptFromFiles(caPath string, privateKeys map[string]string, scriptUri string, scriptParamsUri string, customProcessorDefFactoryInstance CustomProcessorDefFactory, customProcessorsSettings map[string]json.RawMessage) (*ScriptDef, error, ScriptInitProblemType) {
	jsonBytesScript, err := xfer.GetFileBytes(scriptUri, caPath, privateKeys)
	if err != nil {
		return nil, fmt.Errorf("cannot read script: %s", err.Error()), ScriptInitConnectivityProblem
	}
	var jsonBytesParams []byte
	if len(scriptParamsUri) > 0 {
		jsonBytesParams, err = xfer.GetFileBytes(scriptParamsUri, caPath, privateKeys)
		if err != nil {
			return nil, fmt.Errorf("cannot read script parameters: %s", err.Error()), ScriptInitConnectivityProblem
		}
	}

	return NewScriptFromFileBytes(caPath, privateKeys, scriptUri, jsonBytesScript, scriptParamsUri, jsonBytesParams, customProcessorDefFactoryInstance, customProcessorsSettings)
}
