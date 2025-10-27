package mq

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/capillariesio/capillaries/pkg/wfmodel"
)

type CapimqProducer struct {
	url string
}

func (p *CapimqProducer) Open(ctx context.Context, url string, _ string) error {
	p.url = url
	return nil
}

func (p *CapimqProducer) Send(ctx context.Context, msg *wfmodel.Message) error {
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

	bulkRequest = bulkRequest.WithContext(ctx)

	_, claimErr := http.DefaultClient.Do(bulkRequest)
	if claimErr != nil {
		return claimErr
	}
	return nil
}

func (p *CapimqProducer) SendBulk(ctx context.Context, msgs []*wfmodel.Message) error {
	msgsBytes, marshalErr := json.Marshal(msgs)
	if marshalErr != nil {
		return fmt.Errorf("cannot send, error when serializing msgs: %s", marshalErr.Error())
	}

	bulkRequest, bulkReqErr := http.NewRequest(http.MethodPost, p.url+"/q/bulk", bytes.NewReader(msgsBytes))
	if bulkReqErr != nil {
		return bulkReqErr
	}

	bulkRequest.Header.Set("content-type", "application/json")

	bulkRequest = bulkRequest.WithContext(ctx)

	_, claimErr := http.DefaultClient.Do(bulkRequest)
	if claimErr != nil {
		return claimErr
	}
	return nil
}

func (p *CapimqProducer) Close(ctx context.Context) error {
	return nil
}

func (p *CapimqProducer) SupportsSendBulk() bool {
	return true
}
