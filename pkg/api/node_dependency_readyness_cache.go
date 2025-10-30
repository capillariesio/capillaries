package api

import (
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
)

const CachedNodeStateFormat string = "%s %d %d"
const NodeDependencyReadynessCacheMaxElements int = 1000
const NodeDependencyReadynessCacheElementLife time.Duration = 1

var NodeDependencyReadynessCache *expirable.LRU[string, string]

func NewNodeDependencyReadynessCache() *expirable.LRU[string, string] {
	return expirable.NewLRU[string, string](NodeDependencyReadynessCacheMaxElements, nil,
		NodeDependencyReadynessCacheElementLife*time.Second)
}
