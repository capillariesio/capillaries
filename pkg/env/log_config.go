package env

type LogConfig struct {
	Level string `json:"level" env:"CAPI_LOG_LEVEL, overwrite"`
}
