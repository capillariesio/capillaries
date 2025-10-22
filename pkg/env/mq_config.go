package env

type MqConfig struct {
	Port                        int    `json:"mq_port" env:"CAPI_MQ_PORT, overwrite"`                                            // 7654
	AccessControlAllowOrigin    string `json:"access_control_allow_origin" env:"CAPI_MQ_ACCESS_CONTROL_ALLOW_ORIGIN, overwrite"` // http://localhost:8080,http://127.0.0.1:8080
	MaxMessages                 int    `json:"max_messages" env:"CAPI_MQ_MAX_MESSAGES, overwrite"`
	ReturnedDeliveryDelay       int    `json:"returned_delivery_delay" env:"CAPI_MQ_RETURNED_DELIVERY_DELAY, overwrite"`
	DeadAfterNoHeartbeatTimeout int    `json:"dead_after_no_heartbeat_timeout" env:"CAPI_MQ_DEAD_AFER_NO_HEARTBEAT_TIMEOUT, overwrite"`
}
