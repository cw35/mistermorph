package telegramcmd

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	busruntime "github.com/quailyquaily/mistermorph/internal/bus"
)

func TestPublishTelegramBusOutbound(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}))
	bus, err := busruntime.NewInproc(busruntime.InprocOptions{MaxInFlight: 4, Logger: logger})
	if err != nil {
		t.Fatalf("NewInproc() error = %v", err)
	}
	defer bus.Close()

	got := make(chan busruntime.BusMessage, 1)
	if err := bus.Subscribe(busruntime.TopicChatMessage, func(ctx context.Context, msg busruntime.BusMessage) error {
		got <- msg
		return nil
	}); err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}

	messageID, err := publishTelegramBusOutbound(context.Background(), bus, 12345, "hello", "corr:test")
	if err != nil {
		t.Fatalf("publishTelegramBusOutbound() error = %v", err)
	}
	if messageID == "" {
		t.Fatalf("message_id should not be empty")
	}

	select {
	case msg := <-got:
		if msg.Direction != busruntime.DirectionOutbound {
			t.Fatalf("direction mismatch: got %s want %s", msg.Direction, busruntime.DirectionOutbound)
		}
		if msg.Channel != busruntime.ChannelTelegram {
			t.Fatalf("channel mismatch: got %s want %s", msg.Channel, busruntime.ChannelTelegram)
		}
		if msg.Topic != busruntime.TopicChatMessage {
			t.Fatalf("topic mismatch: got %s want %s", msg.Topic, busruntime.TopicChatMessage)
		}
		env, err := msg.Envelope()
		if err != nil {
			t.Fatalf("Envelope() error = %v", err)
		}
		if env.Text != "hello" {
			t.Fatalf("envelope text mismatch: got %q want %q", env.Text, "hello")
		}
		if env.SessionID == "" {
			t.Fatalf("session_id should not be empty")
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("message not delivered")
	}
}

func TestPublishMAEPBusOutbound(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}))
	bus, err := busruntime.NewInproc(busruntime.InprocOptions{MaxInFlight: 4, Logger: logger})
	if err != nil {
		t.Fatalf("NewInproc() error = %v", err)
	}
	defer bus.Close()

	got := make(chan busruntime.BusMessage, 1)
	if err := bus.Subscribe(busruntime.TopicDMReplyV1, func(ctx context.Context, msg busruntime.BusMessage) error {
		got <- msg
		return nil
	}); err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}

	sessionID := "0194e9d5-2f8f-7000-8000-000000000001"
	messageID, err := publishMAEPBusOutbound(context.Background(), bus, "12D3KooWpeerZ", busruntime.TopicDMReplyV1, "reply", sessionID, "", "corr:maep")
	if err != nil {
		t.Fatalf("publishMAEPBusOutbound() error = %v", err)
	}
	if messageID == "" {
		t.Fatalf("message_id should not be empty")
	}

	select {
	case msg := <-got:
		if msg.Direction != busruntime.DirectionOutbound {
			t.Fatalf("direction mismatch: got %s want %s", msg.Direction, busruntime.DirectionOutbound)
		}
		if msg.Channel != busruntime.ChannelMAEP {
			t.Fatalf("channel mismatch: got %s want %s", msg.Channel, busruntime.ChannelMAEP)
		}
		if msg.ParticipantKey != "12D3KooWpeerZ" {
			t.Fatalf("participant_key mismatch: got %q want %q", msg.ParticipantKey, "12D3KooWpeerZ")
		}
		env, err := msg.Envelope()
		if err != nil {
			t.Fatalf("Envelope() error = %v", err)
		}
		if env.Text != "reply" {
			t.Fatalf("envelope text mismatch: got %q want %q", env.Text, "reply")
		}
		if env.SessionID != sessionID {
			t.Fatalf("envelope session_id mismatch: got %q want %q", env.SessionID, sessionID)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("message not delivered")
	}
}
