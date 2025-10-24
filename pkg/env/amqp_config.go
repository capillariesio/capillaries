package env

type Amqp091Config struct {
	URL           string `json:"url" env:"CAPI_AMQP091_URL, overwrite"`                       // RabbitMQ URL, like "amqp://guest:guest@127.0.0.1/" or "amqps://user123:pass123@b-b781b39a-1234-1234-1234-1234fded4f84.mq.us-east-1.on.aws:5671"
	Exchange      string `json:"exchange" env:"CAPI_AMQP091_EXCHANGE, overwrite"`             // Traditionally, "capillaries"
	PrefetchCount int    `json:"prefetch_count" env:"CAPI_AMQP091_PREFETCH_COUNT, overwrite"` // 20
	PrefetchSize  int    `json:"prefetch_size" env:"CAPI_AMQP091_PREFETCH_SIZE, overwrite"`   // 0
	// Became obsolete after we refactored the framework so it sends all batch messages in the very beginning
	// FlowMaxPerConsumer int    `json:"flow_max_per_consumer"`
	// FlowWaitMillisMin  int    `json:"flow_wait_millis_min"`
	// FlowWaitMillisMax  int    `json:"flow_wait_millis_max"`
}

type Amqp10Config struct {
	URL     string `json:"url" env:"CAPI_AMQP10_URL, overwrite"`         // RabbitMQ URL, like "amqp://guest:guest@127.0.0.1/" or "amqps://user123:pass123@b-b781b39a-1234-1234-1234-1234fded4f84.mq.us-east-1.on.aws:5671"
	Address string `json:"address" env:"CAPI_AMQP10_ADDRESS, overwrite"` // Traditionally, "capillaries"
}
