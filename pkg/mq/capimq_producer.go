package mq

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/capillariesio/capillaries/pkg/wfmodel"
)

const CapimqProducerSendTimeout time.Duration = 2000

type CapimqProducer struct {
	url string
}

func NewCapimqProducer(url string) *CapimqProducer {
	return &CapimqProducer{
		url: url,
	}
}

func (p *CapimqProducer) Open() error {
	return nil
}

func (p *CapimqProducer) Send(msg *wfmodel.Message) error {
	msgs := make([]*wfmodel.Message, 1)
	msgs[0] = msg

	msgsBytes, marshalErr := json.Marshal(msgs)
	if marshalErr != nil {
		return fmt.Errorf("cannot send, error when serializing msg: %s", marshalErr.Error())
	}

	bulkRequest, bulkReqErr := http.NewRequest(http.MethodPost, p.url+"/q/bulk", bytes.NewReader(msgsBytes))
	if bulkReqErr != nil {
		return bulkReqErr
	}

	bulkRequest.Header.Set("content-type", "application/json")

	sendCtx, sendCancel := context.WithTimeout(context.Background(), CapimqProducerSendTimeout*time.Millisecond)
	bulkRequest = bulkRequest.WithContext(sendCtx)
	_, claimErr := http.DefaultClient.Do(bulkRequest)
	sendCancel()
	if claimErr != nil {
		return claimErr
	}
	return nil
}

func (p *CapimqProducer) SendBulk(msgs []*wfmodel.Message) error {
	msgsBytes, marshalErr := json.Marshal(msgs)
	if marshalErr != nil {
		return fmt.Errorf("cannot send, error when serializing msgs: %s", marshalErr.Error())
	}

	bulkRequest, bulkReqErr := http.NewRequest(http.MethodPost, p.url+"/q/bulk", bytes.NewReader(msgsBytes))
	if bulkReqErr != nil {
		return bulkReqErr
	}

	bulkRequest.Header.Set("content-type", "application/json")

	sendCtx, sendCancel := context.WithTimeout(context.Background(), CapimqProducerSendTimeout*time.Millisecond)
	bulkRequest = bulkRequest.WithContext(sendCtx)
	_, sendErr := http.DefaultClient.Do(bulkRequest)
	sendCancel()
	if sendErr != nil {
		return sendErr
	}
	return nil
}

func (p *CapimqProducer) Close() error {
	return nil
}

func (p *CapimqProducer) SupportsSendBulk() bool {
	return true
}
