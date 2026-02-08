package agent

import (
	_ "embed"
	"encoding/json"
	"text/template"

	"github.com/quailyquaily/mistermorph/internal/prompttmpl"
	"github.com/quailyquaily/mistermorph/llm"
)

//go:embed prompts/intent_system.tmpl
var intentSystemPromptTemplateSource string

//go:embed prompts/intent_user.tmpl
var intentUserPromptTemplateSource string

var intentPromptTemplateFuncs = template.FuncMap{
	"toJSON": func(v any) (string, error) {
		b, err := json.Marshal(v)
		if err != nil {
			return "", err
		}
		return string(b), nil
	},
}

var intentSystemPromptTemplate = prompttmpl.MustParse("agent_intent_system_prompt", intentSystemPromptTemplateSource, nil)
var intentUserPromptTemplate = prompttmpl.MustParse("agent_intent_user_prompt", intentUserPromptTemplateSource, intentPromptTemplateFuncs)

type intentUserPromptTemplateData struct {
	Task    string
	History []llm.Message
}

func renderIntentPrompts(task string, history []llm.Message) (string, string, error) {
	systemPrompt, err := prompttmpl.Render(intentSystemPromptTemplate, struct{}{})
	if err != nil {
		return "", "", err
	}
	userPrompt, err := prompttmpl.Render(intentUserPromptTemplate, intentUserPromptTemplateData{
		Task:    task,
		History: history,
	})
	if err != nil {
		return "", "", err
	}
	return systemPrompt, userPrompt, nil
}
