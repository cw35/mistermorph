package telegram

import (
	_ "embed"
	"encoding/json"
	"text/template"

	"github.com/quailyquaily/mistermorph/internal/prompttmpl"
)

//go:embed prompts/plan_progress_system.tmpl
var planProgressSystemPromptTemplateSource string

//go:embed prompts/plan_progress_user.tmpl
var planProgressUserPromptTemplateSource string

var planProgressPromptTemplateFuncs = template.FuncMap{
	"toJSON": func(v any) (string, error) {
		b, err := json.Marshal(v)
		if err != nil {
			return "", err
		}
		return string(b), nil
	},
}

var planProgressSystemPromptTemplate = prompttmpl.MustParse("telegram_plan_progress_system", planProgressSystemPromptTemplateSource, nil)
var planProgressUserPromptTemplate = prompttmpl.MustParse("telegram_plan_progress_user", planProgressUserPromptTemplateSource, planProgressPromptTemplateFuncs)

func renderTelegramPlanProgressPrompts(payload map[string]any) (string, string, error) {
	systemPrompt, err := prompttmpl.Render(planProgressSystemPromptTemplate, struct{}{})
	if err != nil {
		return "", "", err
	}
	userPrompt, err := prompttmpl.Render(planProgressUserPromptTemplate, payload)
	if err != nil {
		return "", "", err
	}
	return systemPrompt, userPrompt, nil
}
