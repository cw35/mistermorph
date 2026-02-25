package builtin

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/quailyquaily/mistermorph/llm"
)

type stubPlanCreateLLMClient struct {
	reply string
	err   error
}

func (s *stubPlanCreateLLMClient) Chat(_ context.Context, _ llm.Request) (llm.Result, error) {
	if s.err != nil {
		return llm.Result{}, s.err
	}
	return llm.Result{Text: s.reply}, nil
}

func TestPlanCreateExecuteRejectsEmptyStepsAfterNormalization(t *testing.T) {
	client := &stubPlanCreateLLMClient{
		reply: `{"plan":{"summary":"x","steps":[{"step":"   "},{"step":""}]}}`,
	}
	tool := NewPlanCreateTool(client, "gpt-5.2", []string{"bash"}, 6)
	_, err := tool.Execute(context.Background(), map[string]any{"task": "t"})
	if err == nil {
		t.Fatal("expected error for empty normalized steps")
	}
	if !strings.Contains(err.Error(), "empty steps") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPlanCreateExecuteNormalizesAndDropsEmptySteps(t *testing.T) {
	client := &stubPlanCreateLLMClient{
		reply: `{"plan":{"summary":"x","steps":[{"step":"   ","status":"pending"},{"step":" collect data ","status":""},{"step":"summarize","status":"unknown"}]}}`,
	}
	tool := NewPlanCreateTool(client, "gpt-5.2", []string{"bash"}, 6)
	out, err := tool.Execute(context.Background(), map[string]any{"task": "t"})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	var parsed struct {
		Plan struct {
			Steps []struct {
				Step   string `json:"step"`
				Status string `json:"status"`
			} `json:"steps"`
		} `json:"plan"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(parsed.Plan.Steps) != 2 {
		t.Fatalf("len(steps) = %d, want 2", len(parsed.Plan.Steps))
	}
	if parsed.Plan.Steps[0].Step != "collect data" || parsed.Plan.Steps[0].Status != "in_progress" {
		t.Fatalf("steps[0] = %+v, want step=%q status=%q", parsed.Plan.Steps[0], "collect data", "in_progress")
	}
	if parsed.Plan.Steps[1].Step != "summarize" || parsed.Plan.Steps[1].Status != "pending" {
		t.Fatalf("steps[1] = %+v, want step=%q status=%q", parsed.Plan.Steps[1], "summarize", "pending")
	}
}
