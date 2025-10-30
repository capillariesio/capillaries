package sc

import "github.com/prometheus/client_golang/prometheus"

// Used by Daemon and Webapi
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
