package telegram

import (
	"testing"
	"time"

	"github.com/google/uuid"
	busruntime "github.com/quailyquaily/mistermorph/internal/bus"
)

func TestTelegramOutboundEventFromBusMessage(t *testing.T) {
	sessionID, err := uuid.NewV7()
	if err != nil {
		t.Fatalf("uuid.NewV7() error = %v", err)
	}
	payload, err := busruntime.EncodeMessageEnvelope(busruntime.TopicChatMessage, busruntime.MessageEnvelope{
		MessageID: "msg_1",
		Text:      "hello",
		SentAt:    time.Now().UTC().Format(time.RFC3339),
		SessionID: sessionID.String(),
		ReplyTo:   "123",
	})
	if err != nil {
		t.Fatalf("EncodeMessageEnvelope() error = %v", err)
	}
	event, err := telegramOutboundEventFromBusMessage(busruntime.BusMessage{
		ConversationKey: "tg:42",
		Topic:           busruntime.TopicChatMessage,
		PayloadBase64:   payload,
		CorrelationID:   "telegram:plan:42:1",
		Extensions: busruntime.MessageExtensions{
			ReplyTo: "123",
		},
	})
	if err != nil {
		t.Fatalf("telegramOutboundEventFromBusMessage() error = %v", err)
	}
	if event.ChatID != 42 {
		t.Fatalf("chat id = %d, want 42", event.ChatID)
	}
	if event.ReplyToMessageID != 123 {
		t.Fatalf("reply to message id = %d, want 123", event.ReplyToMessageID)
	}
	if event.Text != "hello" {
		t.Fatalf("text = %q, want hello", event.Text)
	}
	if event.Kind != "plan_progress" {
		t.Fatalf("kind = %q, want plan_progress", event.Kind)
	}
}

func TestTelegramOutboundKind(t *testing.T) {
	if got := telegramOutboundKind("telegram:error:1:2"); got != "error" {
		t.Fatalf("kind(error) = %q, want error", got)
	}
	if got := telegramOutboundKind("telegram:file_download_error:1:2"); got != "error" {
		t.Fatalf("kind(file_download_error) = %q, want error", got)
	}
	if got := telegramOutboundKind("telegram:message:1:2"); got != "message" {
		t.Fatalf("kind(message) = %q, want message", got)
	}
}
