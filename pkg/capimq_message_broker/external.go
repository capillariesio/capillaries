package capimq_message_broker

type CapimqApiGenericResponse struct {
	Data  any    `json:"data"`
	Error string `json:"error"`
}

type CapimqMessage struct {
	Id                   string `json:"id"`
	CapimqWaitRetryGroup string `json:"capimq_wait_retry_group"`
	Data                 []byte `json:"data"`
}

type CapimqApiClaimResponse struct {
	Msg   *CapimqMessage `json:"msg"`
	Error string         `json:"error"`
}
