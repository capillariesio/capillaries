package env

type LogConfig struct {
	LogFile string `json:"log_file" env:"CAPI_LOG_FILE, overwrite"` // If empty, log only to stdout
	Level   string `json:"level" env:"CAPI_LOG_LEVEL, overwrite"`   // zap: DEBUG,INFO,WARN,ERROR,DPANIC,PANIC,FATAL
}
