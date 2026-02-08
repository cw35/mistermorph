package telegramcmd

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/quailyquaily/mistermorph/contacts"
	busruntime "github.com/quailyquaily/mistermorph/internal/bus"
	maepbus "github.com/quailyquaily/mistermorph/internal/bus/adapters/maep"
	telegrambus "github.com/quailyquaily/mistermorph/internal/bus/adapters/telegram"
	"github.com/quailyquaily/mistermorph/maep"
)

func TestBusCrossChannelTelegramInboundToMAEPOutbound(t *testing.T) {
	ctx := context.Background()

	store := contacts.NewFileStore(filepath.Join(t.TempDir(), "contacts"))
	if err := store.Ensure(ctx); err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}))
	inprocBus, err := busruntime.NewInproc(busruntime.InprocOptions{
		MaxInFlight: 8,
		Logger:      logger,
	})
	if err != nil {
		t.Fatalf("NewInproc() error = %v", err)
	}
	defer inprocBus.Close()

	telegramInbound, err := telegrambus.NewInboundAdapter(telegrambus.InboundAdapterOptions{
		Bus:   inprocBus,
		Store: store,
	})
	if err != nil {
		t.Fatalf("NewInboundAdapter(telegram) error = %v", err)
	}

	mockNode := &crossChannelMockNode{
		result: maep.DataPushResult{Accepted: true},
		reqCh:  make(chan maep.DataPushRequest, 1),
	}
	maepDelivery, err := maepbus.NewDeliveryAdapter(maepbus.DeliveryAdapterOptions{
		Node: mockNode,
	})
	if err != nil {
		t.Fatalf("NewDeliveryAdapter(maep) error = %v", err)
	}

	const peerID = "12D3KooWCrossPeer"
	busHandler := func(handlerCtx context.Context, msg busruntime.BusMessage) error {
		switch msg.Direction {
		case busruntime.DirectionInbound:
			if msg.Channel != busruntime.ChannelTelegram {
				return fmt.Errorf("unsupported inbound channel: %s", msg.Channel)
			}
			inbound, err := telegrambus.InboundMessageFromBusMessage(msg)
			if err != nil {
				return err
			}
			sessionID := strings.TrimSpace(msg.Extensions.SessionID)
			if sessionID == "" {
				return fmt.Errorf("session_id is required")
			}
			_, err = publishMAEPBusOutbound(
				handlerCtx,
				inprocBus,
				peerID,
				busruntime.TopicDMReplyV1,
				"ack: "+inbound.Text,
				sessionID,
				msg.Extensions.PlatformMessageID,
				"test:cross_channel",
			)
			return err
		case busruntime.DirectionOutbound:
			if msg.Channel != busruntime.ChannelMAEP {
				return fmt.Errorf("unsupported outbound channel: %s", msg.Channel)
			}
			_, _, err := maepDelivery.Deliver(handlerCtx, msg)
			return err
		default:
			return fmt.Errorf("unsupported direction: %s", msg.Direction)
		}
	}
	for _, topic := range busruntime.AllTopics() {
		if err := inprocBus.Subscribe(topic, busHandler); err != nil {
			t.Fatalf("Subscribe(%s) error = %v", topic, err)
		}
	}

	accepted, err := telegramInbound.HandleInboundMessage(ctx, telegrambus.InboundMessage{
		ChatID:    10001,
		MessageID: 20002,
		ChatType:  "private",
		Text:      "hello from telegram",
	})
	if err != nil {
		t.Fatalf("HandleInboundMessage() error = %v", err)
	}
	if !accepted {
		t.Fatalf("HandleInboundMessage() accepted=false, want true")
	}

	req, got := mockNode.waitRequest(2 * time.Second)
	if !got {
		t.Fatalf("expected MAEP outbound delivery")
	}
	if req.Topic != busruntime.TopicDMReplyV1 {
		t.Fatalf("topic mismatch: got %q want %q", req.Topic, busruntime.TopicDMReplyV1)
	}
	env, err := busruntime.DecodeMessageEnvelope(req.Topic, req.PayloadBase64)
	if err != nil {
		t.Fatalf("DecodeMessageEnvelope() error = %v", err)
	}
	if env.Text != "ack: hello from telegram" {
		t.Fatalf("reply text mismatch: got %q want %q", env.Text, "ack: hello from telegram")
	}
	if strings.TrimSpace(env.SessionID) == "" {
		t.Fatalf("session_id mismatch: expected non-empty")
	}
}

type crossChannelMockNode struct {
	result maep.DataPushResult
	err    error

	reqCh chan maep.DataPushRequest
}

func (m *crossChannelMockNode) PushData(ctx context.Context, peerID string, addresses []string, req maep.DataPushRequest, notification bool) (maep.DataPushResult, error) {
	select {
	case m.reqCh <- req:
	default:
	}
	return m.result, m.err
}

func (m *crossChannelMockNode) waitRequest(timeout time.Duration) (maep.DataPushRequest, bool) {
	select {
	case req := <-m.reqCh:
		return req, true
	case <-time.After(timeout):
		return maep.DataPushRequest{}, false
	}
}
