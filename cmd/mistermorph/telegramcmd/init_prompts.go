package telegramcmd

import (
	_ "embed"
	"encoding/json"
	"text/template"

	"github.com/quailyquaily/mistermorph/internal/prompttmpl"
)

//go:embed prompts/init_questions_system.tmpl
var initQuestionsSystemPromptTemplateSource string

//go:embed prompts/init_questions_user.tmpl
var initQuestionsUserPromptTemplateSource string

//go:embed prompts/init_fill_system.tmpl
var initFillSystemPromptTemplateSource string

//go:embed prompts/init_fill_user.tmpl
var initFillUserPromptTemplateSource string

//go:embed prompts/init_question_message_system.tmpl
var initQuestionMessageSystemPromptTemplateSource string

//go:embed prompts/init_question_message_user.tmpl
var initQuestionMessageUserPromptTemplateSource string

//go:embed prompts/init_post_greeting_system.tmpl
var initPostGreetingSystemPromptTemplateSource string

//go:embed prompts/init_post_greeting_user.tmpl
var initPostGreetingUserPromptTemplateSource string

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
var initQuestionMessageSystemPromptTemplate = prompttmpl.MustParse("telegram_init_question_message_system", initQuestionMessageSystemPromptTemplateSource, nil)
var initQuestionMessageUserPromptTemplate = prompttmpl.MustParse("telegram_init_question_message_user", initQuestionMessageUserPromptTemplateSource, initPromptTemplateFuncs)
var initPostGreetingSystemPromptTemplate = prompttmpl.MustParse("telegram_init_post_greeting_system", initPostGreetingSystemPromptTemplateSource, nil)
var initPostGreetingUserPromptTemplate = prompttmpl.MustParse("telegram_init_post_greeting_user", initPostGreetingUserPromptTemplateSource, initPromptTemplateFuncs)

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

func renderInitQuestionMessagePrompts(payload map[string]any) (string, string, error) {
	systemPrompt, err := prompttmpl.Render(initQuestionMessageSystemPromptTemplate, struct{}{})
	if err != nil {
		return "", "", err
	}
	userPrompt, err := prompttmpl.Render(initQuestionMessageUserPromptTemplate, payload)
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
