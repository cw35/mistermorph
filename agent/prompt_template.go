package agent

import (
	_ "embed"
	"strings"

	"github.com/quailyquaily/mistermorph/internal/prompttmpl"
	"github.com/quailyquaily/mistermorph/tools"
)

//go:embed prompts/system.tmpl
var systemPromptTemplateSource string

var systemPromptTemplate = prompttmpl.MustParse("agent_system_prompt", systemPromptTemplateSource, nil)

type systemPromptTemplateBlock struct {
	Title   string
	Content string
}

type systemPromptTemplateSkill struct {
	Name         string
	FilePath     string
	Description  string
	Requirements []string
}

type systemPromptTemplateData struct {
	Identity      string
	Skills        []systemPromptTemplateSkill
	Blocks        []systemPromptTemplateBlock
	ToolSummaries string
	HasPlanCreate bool
	Rules         []string
}

func renderSystemPrompt(registry *tools.Registry, spec PromptSpec) (string, error) {
	data := systemPromptTemplateData{
		Identity: spec.Identity,
		Skills:   make([]systemPromptTemplateSkill, 0, len(spec.Skills)),
		Blocks:   make([]systemPromptTemplateBlock, 0, len(spec.Blocks)),
		Rules:    make([]string, 0, len(spec.Rules)),
	}
	for _, sk := range spec.Skills {
		name := strings.TrimSpace(sk.Name)
		filePath := strings.TrimSpace(sk.FilePath)
		desc := strings.TrimSpace(sk.Description)
		reqs := make([]string, 0, len(sk.Requirements))
		for _, req := range sk.Requirements {
			req = strings.TrimSpace(req)
			if req == "" {
				continue
			}
			reqs = append(reqs, req)
		}
		if name == "" {
			name = "unknown-skill"
		}
		if filePath == "" {
			filePath = "(unknown)"
		}
		if desc == "" {
			desc = "(not provided)"
		}
		if len(reqs) == 0 {
			reqs = []string{"(not specified)"}
		}
		data.Skills = append(data.Skills, systemPromptTemplateSkill{
			Name:         name,
			FilePath:     filePath,
			Description:  desc,
			Requirements: reqs,
		})
	}
	for _, blk := range spec.Blocks {
		title := strings.TrimSpace(blk.Title)
		if title == "" {
			title = "Context"
		}
		content := strings.TrimSpace(blk.Content)
		if content == "" {
			continue
		}
		data.Blocks = append(data.Blocks, systemPromptTemplateBlock{
			Title:   title,
			Content: content,
		})
	}
	for _, rule := range spec.Rules {
		rule = strings.TrimSpace(rule)
		if rule == "" {
			continue
		}
		data.Rules = append(data.Rules, rule)
	}
	if registry != nil {
		data.ToolSummaries = registry.FormatToolSummaries()
		_, data.HasPlanCreate = registry.Get("plan_create")
	}
	return prompttmpl.Render(systemPromptTemplate, data)
}
