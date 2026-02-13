package chathistory

import (
	"encoding/json"
	"testing"
	"time"
)

func TestRenderContextContent(t *testing.T) {
	out, err := RenderContextContent(ChannelTelegram, []ChatHistoryItem{
		{
			Kind:   KindInboundUser,
			ChatID: "-1001",
			SentAt: time.Date(2026, 2, 13, 10, 0, 0, 0, time.UTC),
			Sender: ChatHistorySender{
				Username:   "alice",
				DisplayRef: "[Alice](tg:@alice)",
			},
			Text: "hello",
			Quote: &ChatHistoryQuote{
				MarkdownBlock: "> [Bob](tg:@bob): hi",
			},
		},
	})
	if err != nil {
		t.Fatalf("RenderContextContent() error = %v", err)
	}
	var payload ContextPayload
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload.Type != ContextType {
		t.Fatalf("type mismatch: got %q want %q", payload.Type, ContextType)
	}
	if payload.Channel != ChannelTelegram {
		t.Fatalf("channel mismatch: got %q want %q", payload.Channel, ChannelTelegram)
	}
	if len(payload.Messages) != 1 {
		t.Fatalf("messages len mismatch: got %d want 1", len(payload.Messages))
	}
	msg := payload.Messages[0]
	if msg.Channel != ChannelTelegram {
		t.Fatalf("item channel mismatch: got %q want %q", msg.Channel, ChannelTelegram)
	}
	if msg.Quote == nil || msg.Quote.MarkdownBlock == "" {
		t.Fatalf("quote markdown block should be present")
	}
}
