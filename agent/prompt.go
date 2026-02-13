package agent

import (
	"strings"

	"github.com/quailyquaily/mistermorph/tools"
)

type PromptSpec struct {
	Identity string
	Rules    []string
	Blocks   []PromptBlock
}

type PromptBlock struct {
	Title   string
	Content string
}

func DefaultPromptSpec() PromptSpec {
	return PromptSpec{
		Identity: "You are MisterMorph, a general-purpose AI agent that can use tools to complete tasks.",
		Rules:    defaultSystemRules(),
	}
}

func BuildSystemPrompt(registry *tools.Registry, spec PromptSpec) string {
	rendered, err := renderSystemPrompt(registry, spec)
	if err == nil && strings.TrimSpace(rendered) != "" {
		return rendered
	}
	return ""
}
