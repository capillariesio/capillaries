package env

type AmqpConfig struct {
	URL           string `json:"url" env:"CAPI_AMQP_URL, overwrite"`
	Exchange      string `json:"exchange" env:"CAPI_AMQP_EXCHANGE, overwrite"`
	PrefetchCount int    `json:"prefetch_count" env:"CAPI_AMQP_PREFETCH_COUNT, overwrite"`
	PrefetchSize  int    `json:"prefetch_size" env:"CAPI_AMQP_PREFETCH_SIZE, overwrite"`
	// Became obsolete after we refactored the framework so it sends all batch messages in the very beginning
	// FlowMaxPerConsumer int    `json:"flow_max_per_consumer"`
	// FlowWaitMillisMin  int    `json:"flow_wait_millis_min"`
	// FlowWaitMillisMax  int    `json:"flow_wait_millis_max"`
}
