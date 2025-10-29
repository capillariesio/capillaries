package env

type Amqp10Config struct {
	URL       string `json:"url" env:"CAPI_AMQP10_URL, overwrite"`               // RabbitMQ URL, like "amqp://guest:guest@127.0.0.1/" or "amqps://user123:pass123@b-b781b39a-1234-1234-1234-1234fded4f84.mq.us-east-1.on.aws:5671"
	Address   string `json:"address" env:"CAPI_AMQP10_ADDRESS, overwrite"`       // Traditionally, "capillaries" for ActiveMQ, "/queue/capidaemon" for RabbitMQ
	AckMethod string `json:"ack_method" env:"CAPI_AMQP10_ACK_METHOD, overwrite"` // release or reject
}
