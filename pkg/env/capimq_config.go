package env

type CapiMqDaemonConfig struct {
	URL               string `json:"url" env:"CAPI_CAPIMQ_URL, overwrite"` // http://localhost:7654
	HeartbeatInterval int64  `json:"heartbeat_interval" env:"CAPI_CAPIMQ_DAEMON_HEARTBEAT_INTERVAL, overwrite"`
}

type CapiMqBrokerConfig struct {
	Port                        int    `json:"mq_port" env:"CAPI_CAPIMQ_PORT, overwrite"`                                                   // 7654
	AccessControlAllowOrigin    string `json:"access_control_allow_origin" env:"CAPI_CAPIMQ_BROKER_ACCESS_CONTROL_ALLOW_ORIGIN, overwrite"` // http://localhost:8080,http://127.0.0.1:8080
	ReturnedDeliveryDelay       int    `json:"returned_delivery_delay" env:"CAPI_CAPIMQ_BROKER_RETURNED_DELIVERY_DELAY, overwrite"`
	MaxMessages                 int    `json:"max_messages" env:"CAPI_CAPIMQ_BROKER_MAX_MESSAGES, overwrite"`
	DeadAfterNoHeartbeatTimeout int    `json:"dead_after_no_heartbeat_timeout" env:"CAPI_CAPIMQ_BROKER_DEAD_AFER_NO_HEARTBEAT_TIMEOUT, overwrite"`
}
