package telegram

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendMessageMarkdownV1ReplyUsesMarkdownParseMode(t *testing.T) {
	var calls []telegramSendMessageRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/bottoken/sendMessage" {
			http.NotFound(w, r)
			return
		}
		var req telegramSendMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		calls = append(calls, req)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	api := newTelegramAPI(srv.Client(), srv.URL, "token")
	err := api.sendMessageMarkdownV1Reply(context.Background(), 42, "*hello*", true, 99)
	if err != nil {
		t.Fatalf("sendMessageMarkdownV1Reply() error = %v", err)
	}
	if len(calls) != 1 {
		t.Fatalf("len(calls) = %d, want 1", len(calls))
	}
	if calls[0].ParseMode != "Markdown" {
		t.Fatalf("parse_mode = %q, want Markdown", calls[0].ParseMode)
	}
	if calls[0].ReplyToMessageID != 99 {
		t.Fatalf("reply_to_message_id = %d, want 99", calls[0].ReplyToMessageID)
	}
}

func TestSendMessageMarkdownV1ReplyFallbackToPlainOnParseError(t *testing.T) {
	var calls []telegramSendMessageRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/bottoken/sendMessage" {
			http.NotFound(w, r)
			return
		}
		var req telegramSendMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		calls = append(calls, req)
		switch req.ParseMode {
		case "Markdown":
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"ok":false,"description":"Bad Request: can't parse entities"}`))
		case "":
			_, _ = w.Write([]byte(`{"ok":true}`))
		default:
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"ok":false,"description":"unexpected parse mode"}`))
		}
	}))
	defer srv.Close()

	api := newTelegramAPI(srv.Client(), srv.URL, "token")
	err := api.sendMessageMarkdownV1Reply(context.Background(), 42, "*bad*", true, 77)
	if err != nil {
		t.Fatalf("sendMessageMarkdownV1Reply() error = %v", err)
	}
	if len(calls) != 2 {
		t.Fatalf("len(calls) = %d, want 2", len(calls))
	}
	if calls[0].ParseMode != "Markdown" || calls[1].ParseMode != "" {
		t.Fatalf("unexpected parse mode sequence: %#v", []string{calls[0].ParseMode, calls[1].ParseMode})
	}
	if calls[0].ReplyToMessageID != 77 || calls[1].ReplyToMessageID != 77 {
		t.Fatalf("reply_to_message_id sequence = %#v, want both 77", []int64{calls[0].ReplyToMessageID, calls[1].ReplyToMessageID})
	}
}
