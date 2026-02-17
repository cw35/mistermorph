package slack

import (
	"testing"
	"time"

	"github.com/google/uuid"
	busruntime "github.com/quailyquaily/mistermorph/internal/bus"
)

func TestSlackOutboundEventFromBusMessage(t *testing.T) {
	sessionID, err := uuid.NewV7()
	if err != nil {
		t.Fatalf("uuid.NewV7() error = %v", err)
	}
	payload, err := busruntime.EncodeMessageEnvelope(busruntime.TopicChatMessage, busruntime.MessageEnvelope{
		MessageID: "msg_1",
		Text:      "hello",
		SentAt:    time.Now().UTC().Format(time.RFC3339),
		SessionID: sessionID.String(),
		ReplyTo:   "1700000000.000100",
	})
	if err != nil {
		t.Fatalf("EncodeMessageEnvelope() error = %v", err)
	}
	event, err := slackOutboundEventFromBusMessage(busruntime.BusMessage{
		ConversationKey: "slack:T111:C222",
		Topic:           busruntime.TopicChatMessage,
		PayloadBase64:   payload,
		CorrelationID:   "slack:error:C222:1700000000.000100",
		Extensions: busruntime.MessageExtensions{
			TeamID:    "T111",
			ChannelID: "C222",
			ThreadTS:  "1700000000.000100",
		},
	})
	if err != nil {
		t.Fatalf("slackOutboundEventFromBusMessage() error = %v", err)
	}
	if event.ConversationKey != "slack:T111:C222" {
		t.Fatalf("conversation key = %q, want slack:T111:C222", event.ConversationKey)
	}
	if event.TeamID != "T111" || event.ChannelID != "C222" {
		t.Fatalf("team/channel = %q/%q, want T111/C222", event.TeamID, event.ChannelID)
	}
	if event.ThreadTS != "1700000000.000100" {
		t.Fatalf("thread ts = %q, want 1700000000.000100", event.ThreadTS)
	}
	if event.Text != "hello" {
		t.Fatalf("text = %q, want hello", event.Text)
	}
	if event.Kind != "error" {
		t.Fatalf("kind = %q, want error", event.Kind)
	}
}

func TestSlackOutboundKind(t *testing.T) {
	if got := slackOutboundKind("slack:error:C:1"); got != "error" {
		t.Fatalf("kind(error) = %q, want error", got)
	}
	if got := slackOutboundKind("slack:message:C:1"); got != "message" {
		t.Fatalf("kind(message) = %q, want message", got)
	}
}
