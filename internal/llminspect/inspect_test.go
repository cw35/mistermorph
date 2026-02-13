package llminspect

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/quailyquaily/mistermorph/llm"
)

func TestModelSceneContext(t *testing.T) {
	if got := ModelSceneFromContext(nil); got != defaultModelScene {
		t.Fatalf("scene for nil ctx = %q, want %q", got, defaultModelScene)
	}
	ctx := WithModelScene(context.Background(), " telegram.addressing_decision ")
	if got := ModelSceneFromContext(ctx); got != "telegram.addressing_decision" {
		t.Fatalf("scene = %q, want telegram.addressing_decision", got)
	}
}

func TestPromptInspectorDumpWithScene(t *testing.T) {
	inspector, err := NewPromptInspector(Options{
		Mode:            "telegram",
		Task:            "telegram",
		TimestampFormat: "20060102_150405",
		DumpDir:         t.TempDir(),
	})
	if err != nil {
		t.Fatalf("NewPromptInspector() error = %v", err)
	}
	if err := inspector.DumpWithScene("telegram.addressing_decision", []llm.Message{
		{Role: "system", Content: "test"},
	}); err != nil {
		t.Fatalf("DumpWithScene() error = %v", err)
	}
	if err := inspector.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	path, content := readSingleDumpFile(t, inspector.file.Name())
	if !strings.Contains(content, "## Request #1") {
		t.Fatalf("dump missing request header: %s", path)
	}
	if !strings.Contains(content, "model_scene: telegram.addressing_decision") {
		t.Fatalf("dump missing model_scene in %s: %q", path, content)
	}
}

func TestPromptClientChatUsesSceneFromContext(t *testing.T) {
	inspector, err := NewPromptInspector(Options{
		Mode:            "telegram",
		Task:            "telegram",
		TimestampFormat: "20060102_150405",
		DumpDir:         t.TempDir(),
	})
	if err != nil {
		t.Fatalf("NewPromptInspector() error = %v", err)
	}
	client := &PromptClient{
		Base:      staticChatClient{},
		Inspector: inspector,
	}
	ctx := WithModelScene(context.Background(), "skills.select")
	_, err = client.Chat(ctx, llm.Request{
		Model: "test-model",
		Messages: []llm.Message{
			{Role: "system", Content: "s"},
			{Role: "user", Content: "u"},
		},
	})
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
	if err := inspector.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	path, content := readSingleDumpFile(t, inspector.file.Name())
	if !strings.Contains(content, "model_scene: skills.select") {
		t.Fatalf("dump missing scene in %s: %q", path, content)
	}
}

func readSingleDumpFile(t *testing.T, path string) (string, string) {
	t.Helper()
	path = strings.TrimSpace(path)
	if path == "" {
		t.Fatalf("empty dump path")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("filepath.Abs() error = %v", err)
	}
	raw, err := os.ReadFile(abs)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	return abs, string(raw)
}

type staticChatClient struct{}

func (staticChatClient) Chat(ctx context.Context, req llm.Request) (llm.Result, error) {
	return llm.Result{Text: `{"type":"final","final":{"output":"ok"}}`}, nil
}
