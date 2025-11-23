package sc

import (
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
)

const ScriptDefCacheMaxElements int = 50
const ScriptDefCacheElementLife time.Duration = 1

// WARING: deserialized ScriptDef can take megabytes (big python scripts, big tag maps), so keep an eye on memory consumption
// If bad comes to worse, implement file-based caching (will require implementing take/restore ScriptDef snapshot code)
var ScriptDefCache *expirable.LRU[string, ScriptInitResult]

func NewScriptDefCache() *expirable.LRU[string, ScriptInitResult] {
	return expirable.NewLRU[string, ScriptInitResult](ScriptDefCacheMaxElements, nil, ScriptDefCacheElementLife*time.Minute)
}
