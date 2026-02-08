package demo

import (
	"context"
	"fmt"
	"time"

	busruntime "github.com/quailyquaily/mistermorph/internal/bus"
	"github.com/quailyquaily/mistermorph/internal/bus/adapters"
)

type InboundOptions struct {
	Bus     *busruntime.Inproc
	Store   adapters.InboundStore
	Channel string
	Now     func() time.Time
}

type InboundAdapter struct {
	flow *adapters.InboundFlow
}

func NewInboundAdapter(opts InboundOptions) (*InboundAdapter, error) {
	flow, err := adapters.NewInboundFlow(adapters.InboundFlowOptions{
		Bus:     opts.Bus,
		Store:   opts.Store,
		Channel: opts.Channel,
		Now:     opts.Now,
	})
	if err != nil {
		return nil, err
	}
	return &InboundAdapter{
		flow: flow,
	}, nil
}

// HandleInbound demonstrates inbox dedupe (channel + platform_message_id)
// before publishing into the internal bus.
func (a *InboundAdapter) HandleInbound(ctx context.Context, platformMessageID string, msg busruntime.BusMessage) (bool, error) {
	if a == nil || a.flow == nil {
		return false, fmt.Errorf("adapter is not initialized")
	}
	return a.flow.PublishValidatedInbound(ctx, platformMessageID, msg)
}
