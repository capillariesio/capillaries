package env

type DaemonConfig struct {
	MessageVerify  MessageVerifyConfig `json:"message_verify,omitempty"`                                // Consumer-side (daemon) signature verification
	ThreadPoolSize int                 `json:"thread_pool_size" env:"CAPI_THREAD_POOL_SIZE, overwrite"` // Daemon threads, like CPUs*1.5
}
