package telegram

import (
	_ "embed"
	"encoding/json"
	"text/template"

	"github.com/quailyquaily/mistermorph/internal/prompttmpl"
)

//go:embed prompts/init_questions_system.md
var initQuestionsSystemPromptTemplateSource string

//go:embed prompts/init_questions_user.md
var initQuestionsUserPromptTemplateSource string

//go:embed prompts/init_fill_system.md
var initFillSystemPromptTemplateSource string

//go:embed prompts/init_fill_user.md
var initFillUserPromptTemplateSource string

//go:embed prompts/init_post_greeting_system.md
var initPostGreetingSystemPromptTemplateSource string

//go:embed prompts/init_post_greeting_user.md
var initPostGreetingUserPromptTemplateSource string

//go:embed prompts/init_soul_polish_system.md
var initSoulPolishSystemPromptTemplateSource string

//go:embed prompts/init_soul_polish_user.md
var initSoulPolishUserPromptTemplateSource string

var initPromptTemplateFuncs = template.FuncMap{
	"toJSON": func(v any) (string, error) {
		b, err := json.Marshal(v)
		if err != nil {
			return "", err
		}
		return string(b), nil
	},
}

var initQuestionsSystemPromptTemplate = prompttmpl.MustParse("telegram_init_questions_system", initQuestionsSystemPromptTemplateSource, nil)
var initQuestionsUserPromptTemplate = prompttmpl.MustParse("telegram_init_questions_user", initQuestionsUserPromptTemplateSource, initPromptTemplateFuncs)
var initFillSystemPromptTemplate = prompttmpl.MustParse("telegram_init_fill_system", initFillSystemPromptTemplateSource, nil)
var initFillUserPromptTemplate = prompttmpl.MustParse("telegram_init_fill_user", initFillUserPromptTemplateSource, initPromptTemplateFuncs)
var initPostGreetingSystemPromptTemplate = prompttmpl.MustParse("telegram_init_post_greeting_system", initPostGreetingSystemPromptTemplateSource, nil)
var initPostGreetingUserPromptTemplate = prompttmpl.MustParse("telegram_init_post_greeting_user", initPostGreetingUserPromptTemplateSource, initPromptTemplateFuncs)

// credits to Peter Steinberger 🦞 https://x.com/steipete/status/2020704611640705485
var initSoulPolishSystemPromptTemplate = prompttmpl.MustParse("telegram_init_soul_polish_system", initSoulPolishSystemPromptTemplateSource, nil)
var initSoulPolishUserPromptTemplate = prompttmpl.MustParse("telegram_init_soul_polish_user", initSoulPolishUserPromptTemplateSource, initPromptTemplateFuncs)

func renderInitQuestionsPrompts(payload map[string]any) (string, string, error) {
	systemPrompt, err := prompttmpl.Render(initQuestionsSystemPromptTemplate, struct{}{})
	if err != nil {
		return "", "", err
	}
	userPrompt, err := prompttmpl.Render(initQuestionsUserPromptTemplate, payload)
	if err != nil {
		return "", "", err
	}
	return systemPrompt, userPrompt, nil
}

func renderInitFillPrompts(payload map[string]any) (string, string, error) {
	systemPrompt, err := prompttmpl.Render(initFillSystemPromptTemplate, struct{}{})
	if err != nil {
		return "", "", err
	}
	userPrompt, err := prompttmpl.Render(initFillUserPromptTemplate, payload)
	if err != nil {
		return "", "", err
	}
	return systemPrompt, userPrompt, nil
}

func renderInitPostGreetingPrompts(payload map[string]any) (string, string, error) {
	systemPrompt, err := prompttmpl.Render(initPostGreetingSystemPromptTemplate, struct{}{})
	if err != nil {
		return "", "", err
	}
	userPrompt, err := prompttmpl.Render(initPostGreetingUserPromptTemplate, payload)
	if err != nil {
		return "", "", err
	}
	return systemPrompt, userPrompt, nil
}

func renderInitSoulPolishPrompts(payload map[string]any) (string, string, error) {
	systemPrompt, err := prompttmpl.Render(initSoulPolishSystemPromptTemplate, struct{}{})
	if err != nil {
		return "", "", err
	}
	userPrompt, err := prompttmpl.Render(initSoulPolishUserPromptTemplate, payload)
	if err != nil {
		return "", "", err
	}
	return systemPrompt, userPrompt, nil
}
