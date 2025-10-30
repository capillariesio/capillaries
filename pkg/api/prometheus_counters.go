package api

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	NodeDependencyReadynessHitCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "capi_daemon_node_dep_ready_cache_hit_count",
		Help: "Capillaries node dependencies readiness cache hits",
	})
	NodeDependencyReadynessMissCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "capi_daemon_node_dep_ready_cache_miss_count",
		Help: "Capillaries node dependencies readiness cache misses",
	})
	NodeDependencyReadynessGetDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "capi_daemon_node_dep_ready_get_duration",
		Help:    "Duration of checkDependencyNodesReady",
		Buckets: prometheus.ExponentialBuckets(0.001, 10.0, 4),
	})
)

var (
	NodeDependencyNoneCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "capi_daemon_node_dep_none_count",
		Help: "Capillaries node dependencies NodeNone count",
	})
	NodeDependencyWaitCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "capi_daemon_node_dep_wait_count",
		Help: "Capillaries node dependencies NodeWait count",
	})
	NodeDependencyGoCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "capi_daemon_node_dep_go_count",
		Help: "Capillaries node dependencies NodeGo count",
	})
	NodeDependencyNogoCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "capi_daemon_node_dep_nogo_count",
		Help: "Capillaries node dependencies NodeNogo count",
	})
)
