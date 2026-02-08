package telegramcmd

import (
	"strings"
	"testing"
)

func TestRenderTelegramPlanProgressPrompts(t *testing.T) {
	payload := map[string]any{
		"task":           "prepare release notes",
		"completed_step": "collect changelog",
		"next_step":      "draft summary",
	}
	sys, user, err := renderTelegramPlanProgressPrompts(payload)
	if err != nil {
		t.Fatalf("renderTelegramPlanProgressPrompts() error = %v", err)
	}
	if !strings.Contains(sys, "casual progress updates for a Telegram chat") {
		t.Fatalf("system prompt missing guidance: %q", sys)
	}
	if !strings.Contains(user, "Generate a progress update for this plan step:") {
		t.Fatalf("user prompt missing instruction: %q", user)
	}
	if !strings.Contains(user, "\"task\":\"prepare release notes\"") {
		t.Fatalf("user prompt missing task payload: %q", user)
	}
}
