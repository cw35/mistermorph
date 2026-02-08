package maep

import (
	"context"
	"fmt"
	"strings"

	busruntime "github.com/quailyquaily/mistermorph/internal/bus"
	maepproto "github.com/quailyquaily/mistermorph/maep"
)

type DataPusher interface {
	PushData(ctx context.Context, peerID string, addresses []string, req maepproto.DataPushRequest, notification bool) (maepproto.DataPushResult, error)
}

type DeliveryAdapterOptions struct {
	Node DataPusher
}

type DeliveryAdapter struct {
	node DataPusher
}

func NewDeliveryAdapter(opts DeliveryAdapterOptions) (*DeliveryAdapter, error) {
	if opts.Node == nil {
		return nil, fmt.Errorf("node is required")
	}
	return &DeliveryAdapter{node: opts.Node}, nil
}

func (a *DeliveryAdapter) Deliver(ctx context.Context, msg busruntime.BusMessage) (bool, bool, error) {
	if a == nil || a.node == nil {
		return false, false, fmt.Errorf("maep delivery adapter is not initialized")
	}
	if ctx == nil {
		return false, false, fmt.Errorf("context is required")
	}
	if msg.Direction != busruntime.DirectionOutbound {
		return false, false, fmt.Errorf("direction must be outbound")
	}
	if msg.Channel != busruntime.ChannelMAEP {
		return false, false, fmt.Errorf("channel must be maep")
	}

	peerID, err := resolvePeerID(msg)
	if err != nil {
		return false, false, err
	}
	req := maepproto.DataPushRequest{
		Topic:          strings.TrimSpace(msg.Topic),
		ContentType:    "application/json",
		PayloadBase64:  strings.TrimSpace(msg.PayloadBase64),
		IdempotencyKey: strings.TrimSpace(msg.IdempotencyKey),
	}
	result, err := a.node.PushData(ctx, peerID, nil, req, false)
	if err != nil {
		return false, false, err
	}
	return result.Accepted, result.Deduped, nil
}
