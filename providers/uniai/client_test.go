package uniai

import (
	"testing"

	"github.com/quailyquaily/mistermorph/llm"
	uniaiapi "github.com/quailyquaily/uniai"
	uniaichat "github.com/quailyquaily/uniai/chat"
)

func TestBuildChatOptionsReplaceMessages(t *testing.T) {
	req := llm.Request{
		Messages: []llm.Message{
			{Role: "user", Content: "new"},
		},
	}

	opts := append(
		[]uniaiapi.ChatOption{uniaiapi.WithMessages(uniaiapi.User("old"))},
		buildChatOptions(req, "", false, uniaiapi.ToolsEmulationOff, nil)...,
	)

	built, err := uniaichat.BuildRequest(opts...)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	if len(built.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(built.Messages))
	}
	if built.Messages[0].Content != "new" {
		t.Fatalf("expected replaced message content 'new', got %q", built.Messages[0].Content)
	}
}

func TestBuildChatOptionsPreserveToolCallIDAsIs(t *testing.T) {
	rawID := "  call_1|ts:abc  "
	req := llm.Request{
		Messages: []llm.Message{
			{
				Role:       "tool",
				Content:    `{"content":"ok"}`,
				ToolCallID: rawID,
			},
		},
	}

	opts := buildChatOptions(req, "", false, uniaiapi.ToolsEmulationOff, nil)
	built, err := uniaichat.BuildRequest(opts...)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	if len(built.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(built.Messages))
	}
	if built.Messages[0].ToolCallID != rawID {
		t.Fatalf("expected tool_call_id preserved as-is %q, got %q", rawID, built.Messages[0].ToolCallID)
	}
}

func TestToolCallThoughtSignatureRoundTrip(t *testing.T) {
	sig := "sig_abc"
	origArgs := `{"path":"/tmp/a.txt","dup":1,"dup":2}`
	orig := []uniaiapi.ToolCall{
		{
			ID:               "call_1",
			Type:             "function",
			ThoughtSignature: sig,
			Function: uniaiapi.ToolCallFunction{
				Name:      "read_file",
				Arguments: origArgs,
			},
		},
	}

	out := toLLMToolCalls(orig)
	if len(out) != 1 {
		t.Fatalf("expected 1 llm tool call, got %d", len(out))
	}
	if out[0].ThoughtSignature != sig {
		t.Fatalf("expected round-tripped thought signature %q, got %q", sig, out[0].ThoughtSignature)
	}
	if out[0].RawArguments != origArgs {
		t.Fatalf("expected raw arguments %q, got %q", origArgs, out[0].RawArguments)
	}

	uniaiCalls := toUniaiToolCallsFromLLM(out)
	if len(uniaiCalls) != 1 {
		t.Fatalf("expected 1 uniai tool call, got %d", len(uniaiCalls))
	}
	if uniaiCalls[0].ThoughtSignature != sig {
		t.Fatalf("expected thought signature %q, got %q", sig, uniaiCalls[0].ThoughtSignature)
	}
	if uniaiCalls[0].Function.Arguments != origArgs {
		t.Fatalf("expected exact raw arguments %q, got %q", origArgs, uniaiCalls[0].Function.Arguments)
	}
}
