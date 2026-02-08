package promptprofile

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/quailyquaily/mistermorph/agent"
	"github.com/quailyquaily/mistermorph/internal/statepaths"
)

const (
	localToolNotesDefaultMaxBytes = 8 * 1024
)

func AppendLocalToolNotesBlock(spec *agent.PromptSpec, log *slog.Logger) {
	if spec == nil {
		return
	}
	if log == nil {
		log = slog.Default()
	}

	path := toolsPath()
	raw, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Warn("prompt_local_tool_notes_load_failed", "path", path, "error", err.Error())
		}
		return
	}

	content := strings.TrimSpace(string(raw))
	if content == "" {
		log.Debug("prompt_local_tool_notes_skipped", "path", path, "reason", "empty")
		return
	}

	maxBytes := localToolNotesMaxBytes()
	content, truncated := truncateUTF8Bytes(content, maxBytes)
	if content == "" {
		log.Debug("prompt_local_tool_notes_skipped", "path", path, "reason", "empty_after_truncate")
		return
	}

	spec.Blocks = append(spec.Blocks, agent.PromptBlock{
		Title:   "Local Tool Notes",
		Content: content,
	})
	log.Info("prompt_local_tool_notes_applied", "path", path, "size", len(content), "max_bytes", maxBytes, "truncated", truncated)
}

func toolsPath() string {
	return filepath.Join(statepaths.FileStateDir(), "TOOLS.md")
}

func localToolNotesMaxBytes() int {
	return localToolNotesDefaultMaxBytes
}

func truncateUTF8Bytes(input string, maxBytes int) (string, bool) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", false
	}
	if maxBytes <= 0 {
		return "", len(input) > 0
	}

	raw := []byte(input)
	if len(raw) <= maxBytes {
		return input, false
	}

	clipped := raw[:maxBytes]
	for len(clipped) > 0 && !utf8.Valid(clipped) {
		clipped = clipped[:len(clipped)-1]
	}
	return strings.TrimSpace(string(clipped)), true
}
