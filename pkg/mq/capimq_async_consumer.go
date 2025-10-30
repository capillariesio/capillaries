package mq

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/capillariesio/capillaries/pkg/l"
	"github.com/capillariesio/capillaries/pkg/wfmodel"
)

const CapimqListenerReceiveTimeout time.Duration = 500
const CapimqListenerNothingToClaimTimeout time.Duration = 500
const CapimqListenerConnErrorTimeout time.Duration = 2000
const CapimqListenerTotalTimeout time.Duration = CapimqListenerReceiveTimeout + CapimqListenerNothingToClaimTimeout + 1000

const CapimqAcknowledgerAckTimeout time.Duration = 2000
const CapimqAcknowledgerTotalTimeout time.Duration = CapimqAcknowledgerAckTimeout + 1000

const CapimqAllProcessorsBusyTimeout time.Duration = 100

type CapimqAsyncConsumer struct {
	url                  string
	clientName           string
	maxProcessors        int
	activeProcessors     atomic.Int64
	listenerStopping     bool
	acknowledgerStopping bool
	listenerDone         chan bool
	acknowledgerDone     chan bool
}

func NewCapimqConsumer(url string, clientName string, maxProcessors int) *CapimqAsyncConsumer {
	return &CapimqAsyncConsumer{
		url:                  url,
		clientName:           clientName,
		maxProcessors:        maxProcessors,
		listenerStopping:     false,
		acknowledgerStopping: false,
	}
}

func (dc *CapimqAsyncConsumer) claim(ctx context.Context) (*wfmodel.Message, error) {
	claimRequest, claimReqErr := http.NewRequest(http.MethodPost, dc.url+"/q/claim", bytes.NewReader([]byte(dc.clientName)))
	if claimReqErr != nil {
		return nil, claimReqErr
	}
	claimRequest.Header.Set("content-type", "plain/text")

	claimResp, claimErr := http.DefaultClient.Do(claimRequest.WithContext(ctx))
	if claimErr != nil {
		return nil, claimErr
	}

	respBody, respErr := io.ReadAll(claimResp.Body)
	if respErr != nil {
		return nil, respErr
	}
	var claimResponse CapimqApiClaimResponse
	if err := json.Unmarshal(respBody, &claimResponse); err != nil {
		return nil, fmt.Errorf("cannot unmarshal CapimqApiClaimResponse, error [%s], body: %s", err.Error(), string(respBody))
	}

	return claimResponse.Data, nil
}

func (dc *CapimqAsyncConsumer) ack(ctx context.Context, msgId string) error {
	ackRequest, ackReqErr := http.NewRequest(http.MethodDelete, dc.url+fmt.Sprintf("/wip/ack/%s", msgId), nil)
	if ackReqErr != nil {
		return ackReqErr
	}
	ackResp, ackErr := http.DefaultClient.Do(ackRequest.WithContext(ctx))
	if ackErr != nil {
		return ackErr
	}
	_, ackRespErr := io.ReadAll(ackResp.Body)
	if ackRespErr != nil {
		return ackRespErr
	}
	return nil
}

func (dc *CapimqAsyncConsumer) retry(ctx context.Context, msgId string) error {
	ackRequest, ackReqErr := http.NewRequest(http.MethodPost, dc.url+fmt.Sprintf("/wip/return/%s", msgId), nil)
	if ackReqErr != nil {
		return ackReqErr
	}
	ackResp, ackErr := http.DefaultClient.Do(ackRequest.WithContext(ctx))
	if ackErr != nil {
		return ackErr
	}
	_, ackRespErr := io.ReadAll(ackResp.Body)
	if ackRespErr != nil {
		return ackRespErr
	}
	return nil
}

func (dc *CapimqAsyncConsumer) heartbeat(ctx context.Context, msgId string) error {
	ackRequest, ackReqErr := http.NewRequest(http.MethodPost, dc.url+fmt.Sprintf("/wip/heartbeat/%s", msgId), nil)
	if ackReqErr != nil {
		return ackReqErr
	}
	ackResp, ackErr := http.DefaultClient.Do(ackRequest.WithContext(ctx))
	if ackErr != nil {
		return ackErr
	}
	_, ackRespErr := io.ReadAll(ackResp.Body)
	if ackRespErr != nil {
		return ackRespErr
	}
	return nil
}

func (dc *CapimqAsyncConsumer) listenerWorker(logger *l.CapiLogger, listenerChannel chan *wfmodel.Message) {
	logger.PushF("CapimqAsyncConsumer.listenerWorker")
	defer logger.PopF()

	for !dc.listenerStopping {
		// Do not be greedy, do not claim that extra message that you are not ready to handle yet anyways
		if int(dc.activeProcessors.Load()) == dc.maxProcessors {
			time.Sleep(CapimqAllProcessorsBusyTimeout * time.Millisecond)
			continue
		}

		claimCtx, claimCancel := context.WithTimeout(context.Background(), CapimqListenerReceiveTimeout*time.Millisecond)
		wfmodelMsg, claimErr := dc.claim(claimCtx)
		claimCancel()
		if claimErr != nil {
			logger.Error("cannot claim: %s", claimErr.Error())
			urlErr := &url.Error{}
			if errors.As(claimErr, &urlErr) {
				time.Sleep(CapimqListenerConnErrorTimeout * time.Millisecond)
			}
			continue
		} else if wfmodelMsg == nil {
			// Nothing to claim
			time.Sleep(CapimqListenerNothingToClaimTimeout * time.Millisecond)
			continue
		}

		dc.activeProcessors.Add(1)
		listenerChannel <- wfmodelMsg
	}

	// Signal that listener is done
	dc.listenerDone <- true
}

func (dc *CapimqAsyncConsumer) acknowledgerWorker(logger *l.CapiLogger, acknowledgerChannel chan AknowledgerToken) {
	logger.PushF("CapimqAsyncConsumer.aknowledgerWorker")
	defer logger.PopF()

	for !dc.acknowledgerStopping {
		timeoutChannel := make(chan bool, 1)
		go func() {
			time.Sleep(1 * time.Second)
			timeoutChannel <- true
		}()
		select {
		// WARNING: make sure the caller does not close acknowledgerChannel before acknowledgerWorker() completes
		case token := <-acknowledgerChannel:
			switch token.Cmd {
			case AcknowledgerCmdAck:
				dc.activeProcessors.Add(-1)
				ackCtx, ackCancel := context.WithTimeout(context.Background(), CapimqAcknowledgerAckTimeout*time.Millisecond)
				ackErr := dc.ack(ackCtx, token.MsgId)
				ackCancel()
				if ackErr != nil {
					// TODO:
					continue
				}
			case AcknowledgerCmdRetry:
				dc.activeProcessors.Add(-1)
				ackCtx, ackCancel := context.WithTimeout(context.Background(), CapimqAcknowledgerAckTimeout*time.Millisecond)
				ackErr := dc.retry(ackCtx, token.MsgId)
				ackCancel()
				if ackErr != nil {
					// TODO:
					continue
				}
			case AcknowledgerCmdHeartbeat:
				ackCtx, ackCancel := context.WithTimeout(context.Background(), CapimqAcknowledgerAckTimeout*time.Millisecond)
				ackErr := dc.heartbeat(ackCtx, token.MsgId)
				ackCancel()
				if ackErr != nil {
					// TODO:
					continue
				}
			}
		case <-timeoutChannel:
			// Biz as usual, check for acknowledgerStopping and select again if still in the loop
		}

	}

	// Signal that Acknowledger is done
	dc.acknowledgerDone <- true
}

func (dc *CapimqAsyncConsumer) Start(logger *l.CapiLogger, listenerChannel chan *wfmodel.Message, acknowledgerChannel chan AknowledgerToken) error {
	listenerLogger, err := l.NewLoggerFromLogger(logger)
	if err != nil {
		return err
	}
	go dc.listenerWorker(listenerLogger, listenerChannel)

	acknowledgererLogger, err := l.NewLoggerFromLogger(logger)
	if err != nil {
		return err
	}
	go dc.acknowledgerWorker(acknowledgererLogger, acknowledgerChannel)

	return nil

}

func (dc *CapimqAsyncConsumer) StopListener(logger *l.CapiLogger) error {
	dc.listenerStopping = true

	timeoutChannel := make(chan bool, 1)
	go func() {
		time.Sleep(CapimqListenerTotalTimeout * time.Second)
		timeoutChannel <- true
	}()
	select {
	case <-dc.listenerDone:
		// Happy path, the caller can close the listener channel
		return nil
	case <-timeoutChannel:
		return fmt.Errorf("cannot stop Listener gracefully, caller closing Listener channel may result in panic")
	}

}

func (dc *CapimqAsyncConsumer) StopAcknowledger(logger *l.CapiLogger) error {
	dc.acknowledgerStopping = true

	timeoutChannel := make(chan bool, 1)
	go func() {
		time.Sleep(CapimqAcknowledgerTotalTimeout * time.Second)
		timeoutChannel <- true
	}()
	select {
	case <-dc.acknowledgerDone:
		// Happy path, the caller can close the Acknowledger channel
		return nil
	case <-timeoutChannel:
		return fmt.Errorf("cannot stop Acknowledger gracefully, caller closing Acknowledger channel may result in panic")
	}

}

func (dc *CapimqAsyncConsumer) SupportsHearbeat() bool {
	return true
}
