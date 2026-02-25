package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type testSendMessageRequest struct {
	ChatID                int64  `json:"chat_id"`
	Text                  string `json:"text"`
	DisableWebPagePreview bool   `json:"disable_web_page_preview"`
}

func TestRunSendSendsFileContentInChunks(t *testing.T) {
	t.Setenv("MISTER_MORPH_TELEGRAM_BOT_TOKEN", "token")
	t.Setenv("TELEGRAM_BOT_TOKEN", "")

	var calls []testSendMessageRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/bottoken/sendMessage" {
			http.NotFound(w, r)
			return
		}
		var req testSendMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		calls = append(calls, req)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	content := strings.Repeat("a", defaultTelegramChunkLen+10)
	path := filepath.Join(t.TempDir(), "verify.txt")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	sent, err := runSend(context.Background(), sendOptions{
		ChatID:     -1001234567890,
		FilePath:   path,
		BaseURL:    srv.URL,
		HTTPClient: srv.Client(),
	})
	if err != nil {
		t.Fatalf("runSend() error = %v", err)
	}
	if sent != 2 {
		t.Fatalf("sent = %d, want 2", sent)
	}
	if len(calls) != 2 {
		t.Fatalf("len(calls) = %d, want 2", len(calls))
	}
	if calls[0].ChatID != -1001234567890 || calls[1].ChatID != -1001234567890 {
		t.Fatalf("chat ids mismatch: %#v", []int64{calls[0].ChatID, calls[1].ChatID})
	}
	if !calls[0].DisableWebPagePreview || !calls[1].DisableWebPagePreview {
		t.Fatalf("disable_web_page_preview must be true")
	}
	if got := calls[0].Text + calls[1].Text; got != content {
		t.Fatalf("merged content mismatch: len(got)=%d len(want)=%d", len(got), len(content))
	}
}

func TestRunSendRequiresToken(t *testing.T) {
	t.Setenv("MISTER_MORPH_TELEGRAM_BOT_TOKEN", "")
	t.Setenv("TELEGRAM_BOT_TOKEN", "")

	path := filepath.Join(t.TempDir(), "verify.txt")
	if err := os.WriteFile(path, []byte("hello"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, err := runSend(context.Background(), sendOptions{
		ChatID:   123,
		FilePath: path,
		BaseURL:  "https://api.telegram.org",
	})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "missing telegram token") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunSendUsesFallbackTokenEnv(t *testing.T) {
	t.Setenv("MISTER_MORPH_TELEGRAM_BOT_TOKEN", "")
	t.Setenv("TELEGRAM_BOT_TOKEN", "token2")

	var called bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.URL.Path != "/bottoken2/sendMessage" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	path := filepath.Join(t.TempDir(), "verify.txt")
	if err := os.WriteFile(path, []byte("hello"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	_, err := runSend(context.Background(), sendOptions{
		ChatID:     123,
		FilePath:   path,
		BaseURL:    srv.URL,
		HTTPClient: srv.Client(),
	})
	if err != nil {
		t.Fatalf("runSend() error = %v", err)
	}
	if !called {
		t.Fatalf("expected telegram API to be called")
	}
}
