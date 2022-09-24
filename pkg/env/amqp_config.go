package env

type AmqpConfig struct {
	URL           string `json:"url"`
	Exchange      string `json:"exchange"`
	PrefetchCount int    `json:"prefetch_count"`
	PrefetchSize  int    `json:"prefetch_size"`
	// Became obsolete after we refactored the framework so it sends all batch messages in the very beginning
	// FlowMaxPerConsumer int    `json:"flow_max_per_consumer"`
	// FlowWaitMillisMin  int    `json:"flow_wait_millis_min"`
	// FlowWaitMillisMax  int    `json:"flow_wait_millis_max"`
}
