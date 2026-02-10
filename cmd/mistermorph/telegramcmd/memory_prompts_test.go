package telegramcmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/quailyquaily/mistermorph/memory"
)

func TestRenderMemoryDraftPrompts(t *testing.T) {
	sys, user, err := renderMemoryDraftPrompts(
		MemoryDraftContext{SessionID: "tg:1", ChatType: "private"},
		[]map[string]string{{"role": "user", "content": "hi"}},
		memory.ShortTermContent{
			SessionSummary: []memory.KVItem{{Title: "Topic", Value: "Users: A"}},
			TemporaryFacts: []memory.KVItem{{Title: "Fact", Value: "URL: https://example.com"}},
		},
	)
	if err != nil {
		t.Fatalf("renderMemoryDraftPrompts() error = %v", err)
	}
	if !strings.Contains(sys, "single agent session") {
		t.Fatalf("unexpected system prompt: %q", sys)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(user), &payload); err != nil {
		t.Fatalf("user prompt is not valid json: %v", err)
	}
	if payload["session_context"] == nil {
		t.Fatalf("missing session_context")
	}
	if payload["rules"] == nil {
		t.Fatalf("missing rules")
	}
	if payload["existing_session_summary"] == nil || payload["existing_temporary_facts"] == nil {
		t.Fatalf("missing existing memory payload")
	}
}

func TestRenderMemoryMergePrompts(t *testing.T) {
	sys, user, err := renderMemoryMergePrompts(
		semanticMergeContent{SessionSummary: []memory.KVItem{{Title: "A", Value: "v"}}},
		semanticMergeContent{TemporaryFacts: []memory.KVItem{{Title: "B", Value: "v"}}},
	)
	if err != nil {
		t.Fatalf("renderMemoryMergePrompts() error = %v", err)
	}
	if !strings.Contains(sys, "merge short-term memory entries") {
		t.Fatalf("unexpected system prompt: %q", sys)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(user), &payload); err != nil {
		t.Fatalf("user prompt is not valid json: %v", err)
	}
	if payload["existing"] == nil || payload["incoming"] == nil {
		t.Fatalf("missing existing/incoming payload")
	}
}
