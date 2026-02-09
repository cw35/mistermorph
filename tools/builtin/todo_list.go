package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/quailyquaily/mistermorph/internal/pathutil"
	"github.com/quailyquaily/mistermorph/internal/todo"
)

type TodoListTool struct {
	Enabled  bool
	WIPPath  string
	DONEPath string
}

func NewTodoListTool(enabled bool, wipPath string, donePath string) *TodoListTool {
	return &TodoListTool{
		Enabled:  enabled,
		WIPPath:  strings.TrimSpace(wipPath),
		DONEPath: strings.TrimSpace(donePath),
	}
}

func (t *TodoListTool) Name() string { return "todo_list" }

func (t *TodoListTool) Description() string {
	return "Lists current TODO items from TODO.WIP.md and TODO.DONE.md."
}

func (t *TodoListTool) ParameterSchema() string {
	s := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"scope": map[string]any{
				"type":        "string",
				"description": "Which items to return: wip|done|both (default wip).",
			},
		},
	}
	b, _ := json.MarshalIndent(s, "", "  ")
	return string(b)
}

func (t *TodoListTool) Execute(_ context.Context, params map[string]any) (string, error) {
	if !t.Enabled {
		return "", fmt.Errorf("todo_list tool is disabled")
	}
	wipPath := pathutil.ExpandHomePath(strings.TrimSpace(t.WIPPath))
	donePath := pathutil.ExpandHomePath(strings.TrimSpace(t.DONEPath))
	if wipPath == "" || donePath == "" {
		return "", fmt.Errorf("todo paths are not configured")
	}
	scope, _ := params["scope"].(string)
	scope = strings.ToLower(strings.TrimSpace(scope))
	store := todo.NewStore(wipPath, donePath)
	result, err := store.List(scope)
	if err != nil {
		return "", err
	}
	out, _ := json.MarshalIndent(result, "", "  ")
	return string(out), nil
}
