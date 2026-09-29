package env

type DaemonConfig struct {
	MessageVerify  MessageVerifyConfig `json:"message_verify,omitempty"`                                // Consumer-side (daemon) signature verification
	ThreadPoolSize int                 `json:"thread_pool_size" env:"CAPI_THREAD_POOL_SIZE, overwrite"` // Daemon threads, like CPUs*1.5
	// Rel lookup: number of goroutines processing lookup keys (partition key "key") in parallel. Default 20.
	LookupKeyWorkers int `json:"lookup_key_workers" env:"CAPI_LOOKUP_KEY_WORKERS, overwrite"`
	// Rel lookup: number of goroutines within each key worker retrieving data rows by rowid in parallel. Default 10.
	LookupRowidWorkers int `json:"lookup_rowid_workers" env:"CAPI_LOOKUP_ROWID_WORKERS, overwrite"`
}
