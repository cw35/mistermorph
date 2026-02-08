package telegramcmd

import (
	"strings"
	"testing"
)

func TestRenderInitQuestionsPrompts(t *testing.T) {
	payload := map[string]any{
		"user_text": "hello",
		"required_targets": map[string]any{
			"identity": []string{"Name", "Creature", "Vibe", "Emoji"},
			"soul":     []string{"Core Truths", "Boundaries", "Vibe"},
		},
	}
	sys, user, err := renderInitQuestionsPrompts(payload)
	if err != nil {
		t.Fatalf("renderInitQuestionsPrompts() error = %v", err)
	}
	if !strings.Contains(sys, "Return JSON only") {
		t.Fatalf("system prompt missing contract: %q", sys)
	}
	if !strings.Contains(sys, "\"message\"") {
		t.Fatalf("system prompt missing message contract: %q", sys)
	}
	if !strings.Contains(user, "\"user_text\":\"hello\"") {
		t.Fatalf("user prompt missing payload: %q", user)
	}
}

func TestRenderInitFillPrompts(t *testing.T) {
	payload := map[string]any{
		"user_answer": "I want you to be concise.",
	}
	sys, user, err := renderInitFillPrompts(payload)
	if err != nil {
		t.Fatalf("renderInitFillPrompts() error = %v", err)
	}
	if !strings.Contains(sys, "\"identity\"") || !strings.Contains(sys, "\"soul\"") {
		t.Fatalf("system prompt missing schema: %q", sys)
	}
	if !strings.Contains(user, "\"user_answer\":\"I want you to be concise.\"") {
		t.Fatalf("user prompt missing payload: %q", user)
	}
}

func TestRenderInitPostGreetingPrompts(t *testing.T) {
	payload := map[string]any{
		"identity_markdown": "# IDENTITY",
		"soul_markdown":     "# SOUL",
	}
	sys, user, err := renderInitPostGreetingPrompts(payload)
	if err != nil {
		t.Fatalf("renderInitPostGreetingPrompts() error = %v", err)
	}
	if !strings.Contains(sys, "persona bootstrap") {
		t.Fatalf("system prompt missing context: %q", sys)
	}
	if !strings.Contains(user, "\"identity_markdown\":\"# IDENTITY\"") {
		t.Fatalf("user prompt missing payload: %q", user)
	}
}
