package agent

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/quailyquaily/mistermorph/llm"
)

func TestRenderIntentPrompts_UserPayloadIsJSON(t *testing.T) {
	task := "Hi"
	history := []llm.Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi"},
	}

	sys, user, err := renderIntentPrompts(task, history)
	if err != nil {
		t.Fatalf("renderIntentPrompts() error = %v", err)
	}
	if !strings.Contains(sys, "Return ONLY JSON") {
		t.Fatalf("system prompt missing schema guard: %q", sys)
	}

	var payload struct {
		Task    string        `json:"task"`
		History []llm.Message `json:"history"`
		Rules   []string      `json:"rules"`
	}
	if err := json.Unmarshal([]byte(user), &payload); err != nil {
		t.Fatalf("user prompt is not valid json: %v", err)
	}
	if payload.Task != task {
		t.Fatalf("payload.task = %q, want %q", payload.Task, task)
	}
	if len(payload.History) != len(history) {
		t.Fatalf("payload.history len = %d, want %d", len(payload.History), len(history))
	}
	if len(payload.Rules) == 0 {
		t.Fatalf("payload.rules is empty")
	}
	if !strings.Contains(strings.Join(payload.Rules, "\n"), "question and request are independent") {
		t.Fatalf("payload.rules missing expected rule")
	}
}
