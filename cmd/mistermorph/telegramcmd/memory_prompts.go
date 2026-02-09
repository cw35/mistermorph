package telegramcmd

import (
	_ "embed"
	"encoding/json"
	"text/template"

	"github.com/quailyquaily/mistermorph/internal/prompttmpl"
	"github.com/quailyquaily/mistermorph/memory"
)

//go:embed prompts/memory_draft_system.tmpl
var memoryDraftSystemPromptTemplateSource string

//go:embed prompts/memory_draft_user.tmpl
var memoryDraftUserPromptTemplateSource string

//go:embed prompts/memory_merge_system.tmpl
var memoryMergeSystemPromptTemplateSource string

//go:embed prompts/memory_merge_user.tmpl
var memoryMergeUserPromptTemplateSource string

var memoryPromptTemplateFuncs = template.FuncMap{
	"toJSON": func(v any) (string, error) {
		b, err := json.Marshal(v)
		if err != nil {
			return "", err
		}
		return string(b), nil
	},
}

var memoryDraftSystemPromptTemplate = prompttmpl.MustParse("telegram_memory_draft_system", memoryDraftSystemPromptTemplateSource, nil)
var memoryDraftUserPromptTemplate = prompttmpl.MustParse("telegram_memory_draft_user", memoryDraftUserPromptTemplateSource, memoryPromptTemplateFuncs)
var memoryMergeSystemPromptTemplate = prompttmpl.MustParse("telegram_memory_merge_system", memoryMergeSystemPromptTemplateSource, nil)
var memoryMergeUserPromptTemplate = prompttmpl.MustParse("telegram_memory_merge_user", memoryMergeUserPromptTemplateSource, memoryPromptTemplateFuncs)

type memoryDraftUserPromptData struct {
	SessionContext         MemoryDraftContext
	Conversation           []map[string]string
	ExistingSessionSummary []memory.KVItem
	ExistingTemporaryFacts []memory.KVItem
}

func renderMemoryDraftPrompts(
	ctxInfo MemoryDraftContext,
	conversation []map[string]string,
	existing memory.ShortTermContent,
) (string, string, error) {
	systemPrompt, err := prompttmpl.Render(memoryDraftSystemPromptTemplate, struct{}{})
	if err != nil {
		return "", "", err
	}
	userPrompt, err := prompttmpl.Render(memoryDraftUserPromptTemplate, memoryDraftUserPromptData{
		SessionContext:         ctxInfo,
		Conversation:           conversation,
		ExistingSessionSummary: existing.SessionSummary,
		ExistingTemporaryFacts: existing.TemporaryFacts,
	})
	if err != nil {
		return "", "", err
	}
	return systemPrompt, userPrompt, nil
}

type memoryMergeUserPromptData struct {
	Existing semanticMergeContent
	Incoming semanticMergeContent
}

func renderMemoryMergePrompts(existing semanticMergeContent, incoming semanticMergeContent) (string, string, error) {
	systemPrompt, err := prompttmpl.Render(memoryMergeSystemPromptTemplate, struct{}{})
	if err != nil {
		return "", "", err
	}
	userPrompt, err := prompttmpl.Render(memoryMergeUserPromptTemplate, memoryMergeUserPromptData{
		Existing: existing,
		Incoming: incoming,
	})
	if err != nil {
		return "", "", err
	}
	return systemPrompt, userPrompt, nil
}
