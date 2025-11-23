package env

type DaemonConfig struct {
	ThreadPoolSize int `json:"thread_pool_size" env:"CAPI_THREAD_POOL_SIZE, overwrite"` // Daemon threads, like CPUs*1.5
}
