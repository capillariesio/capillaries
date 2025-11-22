package capimq_message_broker

type CapimqMessage struct {
	Id                   string `json:"id"`
	CapimqWaitRetryGroup string `json:"capimq_wait_retry_group"` // Used by producer and CapiMQ, not used by consumer
	Data                 []byte `json:"data"`
}

type CapimqResultType interface {
	int | *CapimqMessage | []*CapimqMessage
}
