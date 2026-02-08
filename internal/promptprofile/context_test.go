package promptprofile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/quailyquaily/mistermorph/agent"
	"github.com/spf13/viper"
)

func TestAppendLocalToolNotesBlock_MissingFileNoChange(t *testing.T) {
	stateDir := t.TempDir()
	prevStateDir := viper.GetString("file_state_dir")
	viper.Set("file_state_dir", stateDir)
	t.Cleanup(func() {
		viper.Set("file_state_dir", prevStateDir)
	})

	spec := agent.DefaultPromptSpec()
	AppendLocalToolNotesBlock(&spec, nil)
	if len(spec.Blocks) != 0 {
		t.Fatalf("expected no blocks when TOOLS.md is missing")
	}
}

func TestAppendLocalToolNotesBlock_EmptyFileSkipped(t *testing.T) {
	stateDir := t.TempDir()
	prevStateDir := viper.GetString("file_state_dir")
	viper.Set("file_state_dir", stateDir)
	t.Cleanup(func() {
		viper.Set("file_state_dir", prevStateDir)
	})

	if err := os.WriteFile(filepath.Join(stateDir, "TOOLS.md"), []byte(" \n\t "), 0o644); err != nil {
		t.Fatalf("write TOOLS.md: %v", err)
	}

	spec := agent.DefaultPromptSpec()
	AppendLocalToolNotesBlock(&spec, nil)
	if len(spec.Blocks) != 0 {
		t.Fatalf("expected no blocks for empty TOOLS.md")
	}
}

func TestAppendLocalToolNotesBlock_AppendsContent(t *testing.T) {
	stateDir := t.TempDir()
	prevStateDir := viper.GetString("file_state_dir")
	viper.Set("file_state_dir", stateDir)
	t.Cleanup(func() {
		viper.Set("file_state_dir", prevStateDir)
	})

	content := "- run: go test ./...\n- cache: ./tmp\n"
	if err := os.WriteFile(filepath.Join(stateDir, "TOOLS.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write TOOLS.md: %v", err)
	}

	spec := agent.DefaultPromptSpec()
	AppendLocalToolNotesBlock(&spec, nil)
	if len(spec.Blocks) != 1 {
		t.Fatalf("expected one block, got %d", len(spec.Blocks))
	}
	if spec.Blocks[0].Title != "Local Tool Notes" {
		t.Fatalf("unexpected block title: %q", spec.Blocks[0].Title)
	}
	if !strings.Contains(spec.Blocks[0].Content, "go test ./...") {
		t.Fatalf("unexpected block content: %q", spec.Blocks[0].Content)
	}
}

func TestAppendLocalToolNotesBlock_TruncatesByFixedMaxBytes(t *testing.T) {
	stateDir := t.TempDir()
	prevStateDir := viper.GetString("file_state_dir")
	viper.Set("file_state_dir", stateDir)
	t.Cleanup(func() {
		viper.Set("file_state_dir", prevStateDir)
	})

	content := strings.Repeat("a", localToolNotesDefaultMaxBytes+100)
	if err := os.WriteFile(filepath.Join(stateDir, "TOOLS.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write TOOLS.md: %v", err)
	}

	spec := agent.DefaultPromptSpec()
	AppendLocalToolNotesBlock(&spec, nil)
	if len(spec.Blocks) != 1 {
		t.Fatalf("expected one block, got %d", len(spec.Blocks))
	}
	if got := spec.Blocks[0].Content; got != strings.Repeat("a", localToolNotesDefaultMaxBytes) {
		t.Fatalf("unexpected truncated content: %q", got)
	}
}
