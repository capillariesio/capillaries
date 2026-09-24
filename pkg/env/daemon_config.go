package env

type DaemonConfig struct {
	MessageVerify  MessageVerifyConfig `json:"message_verify,omitempty"`                                // Consumer-side (daemon) signature verification
	ThreadPoolSize int                 `json:"thread_pool_size" env:"CAPI_THREAD_POOL_SIZE, overwrite"` // Daemon threads, like CPUs*1.5
	// Max number of partition keys in IN clause of SELECT, used for lookups an rowid searches,
	// Default 5, keep it <10 to avoid Cassandra quorum mechanism kicked in.
	// Amazon Keyspaces allows up to 100
	MaxPartitionKeysInSelect int `json:"max_partition_keys_in_select" env:"CAPI_MAX_PARTITION_KEYS_IN_SELECT, overwrite"`
}
