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

func NewScriptFromFiles(fetchPolicy *FetchPolicy, caPath string, privateKeys map[string]string, scriptUrl string, scriptParamsUrl string, customProcessorDefFactoryInstance CustomProcessorDefFactory, customProcessorsSettings map[string]json.RawMessage) (*ScriptDef, ScriptInitProblemType, error) {

	// Gate externally supplied script/params URLs before fetching anything (SSRF / local-file disclosure hardening).
	// A nil or unconfigured policy allows everything (legacy behavior).
	if err := fetchPolicy.CheckUrl(scriptUrl); err != nil {
		return nil, ScriptInitConnectivityProblem, fmt.Errorf("cannot read script %s: %s", scriptUrl, err.Error())
	}
	if scriptParamsUrl != "" {
		if err := fetchPolicy.CheckUrl(scriptParamsUrl); err != nil {
			return nil, ScriptInitConnectivityProblem, fmt.Errorf("cannot read script parameters %s: %s", scriptParamsUrl, err.Error())
		}
	}

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

	// Gate the file URLs embedded in the script itself - file_reader "urls" and file_creator
	// "url_template" - with the same policy that guards the top-level script/params URLs above.
	// Without this, the top-level gate only vets where the script came from, not the local paths
	// and network endpoints the script then makes the daemon read from and write to (SSRF /
	// local-file read / arbitrary-file overwrite). Only meaningful once we have a fully parsed def.
	if err == nil && initProblem == ScriptInitNoProblem && scriptDef != nil {
		if urlErr := checkNodeFileUrls(fetchPolicy, scriptDef); urlErr != nil {
			// Treat a policy violation like the top-level URL gate: reject and do not cache.
			return nil, ScriptInitConnectivityProblem, urlErr
		}
	}

	if ScriptDefCache != nil && initProblem != ScriptInitConnectivityProblem {
		ScriptDefCache.Add(scriptCacheKey, ScriptInitResult{scriptDef, initProblem, err})
	}
	return scriptDef, initProblem, err
}

// checkNodeFileUrls applies fetchPolicy to every file URL a script would make the daemon touch at
// run time: each file_reader source url ("urls") and each file_creator destination ("url_template").
// url_template may still contain {run_id}/{batch_idx} placeholders at this point; those live in the
// path portion of the URL, so scheme/host gating is unaffected. A nil/unconfigured policy allows
// everything (legacy behavior), matching FetchPolicy.CheckUrl.
func checkNodeFileUrls(fetchPolicy *FetchPolicy, scriptDef *ScriptDef) error {
	for nodeName, node := range scriptDef.ScriptNodes {
		if node.HasFileReader() {
			for _, srcUrl := range node.FileReader.SrcFileUrls {
				if err := fetchPolicy.CheckUrl(srcUrl); err != nil {
					return fmt.Errorf("node %s: file reader source url %s is not allowed: %s", nodeName, srcUrl, err.Error())
				}
			}
		}
		if node.HasFileCreator() {
			if err := fetchPolicy.CheckUrl(node.FileCreator.UrlTemplate); err != nil {
				return fmt.Errorf("node %s: file creator url_template %s is not allowed: %s", nodeName, node.FileCreator.UrlTemplate, err.Error())
			}
		}
	}
	return nil
}
