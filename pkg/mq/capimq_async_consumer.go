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

	"github.com/capillariesio/capillaries/pkg/capimq_message_broker"
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
	req, reqErr := http.NewRequest(http.MethodPost, dc.url+"/q/claim", bytes.NewReader([]byte(dc.clientName)))
	if reqErr != nil {
		return nil, reqErr
	}
	req.Header.Set("content-type", "plain/text")

	resp, bodyErr := http.DefaultClient.Do(req.WithContext(ctx))
	if bodyErr != nil {
		return nil, bodyErr
	}
	defer resp.Body.Close()

	bodyBytes, bodyErr := io.ReadAll(resp.Body)
	if bodyErr != nil {
		return nil, bodyErr
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cannot claim, HTTP response %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var claimResponse capimq_message_broker.CapimqApiClaimResponse
	if err := json.Unmarshal(bodyBytes, &claimResponse); err != nil {
		return nil, fmt.Errorf("cannot unmarshal CapimqApiClaimResponse, error [%s], body: %s", err.Error(), string(bodyBytes))
	}

	var wfmodelMsg wfmodel.Message
	if err := json.Unmarshal(claimResponse.Msg.Data, &wfmodelMsg); err != nil {
		return nil, fmt.Errorf("cannot unmarshal CapimqApiClaimResponse.Msg.Data, error [%s], body: %s", err.Error(), string(claimResponse.Msg.Data))
	}

	// Copy id just in case capimq broker overrode it (it should not!)
	wfmodelMsg.Id = claimResponse.Msg.Id

	return &wfmodelMsg, nil
}

func (dc *CapimqAsyncConsumer) ack(ctx context.Context, msgId string) error {
	req, reqErr := http.NewRequest(http.MethodDelete, dc.url+fmt.Sprintf("/wip/ack/%s", msgId), nil)
	if reqErr != nil {
		return reqErr
	}
	resp, respErr := http.DefaultClient.Do(req.WithContext(ctx))
	if respErr != nil {
		return respErr
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("cannot send ack, HTTP response %d: %s", resp.StatusCode, string(bodyBytes))
	}
	return nil
}

func (dc *CapimqAsyncConsumer) retry(ctx context.Context, msgId string) error {
	req, reqErr := http.NewRequest(http.MethodPost, dc.url+fmt.Sprintf("/wip/return/%s", msgId), nil)
	if reqErr != nil {
		return reqErr
	}
	resp, respErr := http.DefaultClient.Do(req.WithContext(ctx))
	if respErr != nil {
		return respErr
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("cannot send retry, HTTP response %d: %s", resp.StatusCode, string(bodyBytes))
	}
	return nil
}

func (dc *CapimqAsyncConsumer) heartbeat(ctx context.Context, msgId string) error {
	req, reqErr := http.NewRequest(http.MethodPost, dc.url+fmt.Sprintf("/wip/heartbeat/%s", msgId), nil)
	if reqErr != nil {
		return reqErr
	}
	resp, respErr := http.DefaultClient.Do(req.WithContext(ctx))
	if respErr != nil {
		return respErr
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("cannot send heartbeat, HTTP response %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

func (dc *CapimqAsyncConsumer) listenerWorker(logger *l.CapiLogger, listenerChannel chan *wfmodel.Message) {
	logger.PushF("CapimqAsyncConsumer.listenerWorker")
	defer logger.Close()

	for !dc.listenerStopping {
		// Do not claim until at least one procesor is ready, otherwise we risk a msg sitting
		// in the channel without sending heartbits, so by the time a processor start handling it,
		// CapiMQ already may consider it dead
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
	defer logger.Close()

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
				ackCtx, ackCancel := context.WithTimeout(context.Background(), CapimqAcknowledgerAckTimeout*time.Millisecond)
				ackErr := dc.ack(ackCtx, token.MsgId)
				ackCancel()
				if ackErr != nil {
					logger.Error("cannot send ack, msg %s: %s", token.MsgId, ackErr.Error())
					continue
				}
			case AcknowledgerCmdRetry:
				ackCtx, ackCancel := context.WithTimeout(context.Background(), CapimqAcknowledgerAckTimeout*time.Millisecond)
				ackErr := dc.retry(ackCtx, token.MsgId)
				ackCancel()
				if ackErr != nil {
					logger.Error("cannot send retry, msg %s: %s", token.MsgId, ackErr.Error())
					continue
				}
			case AcknowledgerCmdHeartbeat:
				ackCtx, ackCancel := context.WithTimeout(context.Background(), CapimqAcknowledgerAckTimeout*time.Millisecond)
				ackErr := dc.heartbeat(ackCtx, token.MsgId)
				ackCancel()
				if ackErr != nil {
					logger.Error("cannot send heartbeat, msg %s: %s", token.MsgId, ackErr.Error())
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

	acknowledgerLogger, err := l.NewLoggerFromLogger(logger)
	if err != nil {
		return err
	}
	go dc.acknowledgerWorker(acknowledgerLogger, acknowledgerChannel)

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

func (dc *CapimqAsyncConsumer) DecrementActiveProcessors() {
	dc.activeProcessors.Add(-1)
}
