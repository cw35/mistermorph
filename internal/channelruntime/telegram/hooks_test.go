package telegram

import (
	"context"
	"io"
	"log/slog"
	"testing"
)

func TestHooksRecoverPanic(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	callInboundHook(context.Background(), logger, Hooks{
		OnInbound: func(context.Context, InboundEvent) {
			panic("inbound panic")
		},
	}, InboundEvent{})

	callOutboundHook(context.Background(), logger, Hooks{
		OnOutbound: func(context.Context, OutboundEvent) {
			panic("outbound panic")
		},
	}, OutboundEvent{})

	callErrorHook(context.Background(), logger, Hooks{
		OnError: func(context.Context, ErrorEvent) {
			panic("error panic")
		},
	}, ErrorEvent{Err: context.Canceled})
}
