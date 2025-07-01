package sc

import (
	"encoding/json"
	"fmt"

	"github.com/capillariesio/capillaries/pkg/xfer"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/prometheus/client_golang/prometheus"
)

type ScriptInitResult struct {
	Def         *ScriptDef
	InitProblem ScriptInitProblemType
	Err         error
}

// WARING: deserialized ScriptDef can take megabytes (big python scripts, big tag maps), so keep an eye on memory consumption
// If bad comes to worse, implement file-based caching (will require implementing take/restore ScriptDef snapshot code)
var ScriptDefCache *expirable.LRU[string, ScriptInitResult]
var (
	ScriptDefCacheHitCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "capi_script_def_cache_hit_count",
		Help: "Capillaries script def cache hits",
	})
	ScriptDefCacheMissCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "capi_script_def_cache_miss_count",
		Help: "Capillaries script def cache miss",
	})
)

func NewScriptFromFiles(caPath string, privateKeys map[string]string, scriptUrl string, scriptParamsUrl string, customProcessorDefFactoryInstance CustomProcessorDefFactory, customProcessorsSettings map[string]json.RawMessage) (*ScriptDef, ScriptInitProblemType, error) {

	scriptCacheKey := fmt.Sprintf("%s %s", scriptUrl, scriptParamsUrl)
	if ScriptDefCache != nil {
		if cachedScriptInitResult, ok := ScriptDefCache.Get(scriptCacheKey); ok {
			ScriptDefCacheHitCounter.Inc()
			return cachedScriptInitResult.Def, cachedScriptInitResult.InitProblem, cachedScriptInitResult.Err
		} else {
			ScriptDefCacheMissCounter.Inc()
		}
	}

	var err error
	var jsonBytesScript []byte
	var jsonBytesParams []byte

	if jsonBytesScript == nil {
		jsonBytesScript, err = xfer.GetFileBytes(scriptUrl, caPath, privateKeys)
		if err != nil {
			return nil, ScriptInitConnectivityProblem, fmt.Errorf("cannot read script %s: %s", scriptUrl, err.Error())
		}
	}

	if jsonBytesParams == nil && scriptParamsUrl != "" {
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
