package mq

import (
	"context"

	"github.com/capillariesio/capillaries/pkg/wfmodel"
)

type MqProducer interface {
	Open(ctx context.Context, url string, address string) error
	Close(ctx context.Context) error
	Send(ctx context.Context, msg *wfmodel.Message) error
}
