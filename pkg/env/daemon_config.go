package env

type DaemonConfig struct {
	ThreadPoolSize int `json:"thread_pool_size" env:"CAPI_THREAD_POOL_SIZE, overwrite"`
	DeadLetterTtl  int `json:"dead_letter_ttl" env:"CAPI_DEAD_LETTER_TTL, overwrite"`
}
