package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/quailyquaily/mistermorph/agent"
	"github.com/quailyquaily/mistermorph/cmd/telegram"
	"github.com/quailyquaily/mistermorph/llm"
	"github.com/quailyquaily/mistermorph/memory"
	"github.com/spf13/cobra"
)

func newTelegramCommand() *cobra.Command {
	return telegram.NewCommand(telegram.Dependencies{
		LoggerFromViper:     loggerFromViper,
		LogOptionsFromViper: logOptionsFromViper,
		CreateLLMClient: func(provider, endpoint, apiKey, model string, timeout time.Duration) (llm.Client, error) {
			return llmClientFromConfig(llmClientConfig{
				Provider:       provider,
				Endpoint:       endpoint,
				APIKey:         apiKey,
				Model:          model,
				RequestTimeout: timeout,
			})
		},
		LLMProviderFromViper:   llmProviderFromViper,
		LLMEndpointForProvider: llmEndpointForProvider,
		LLMAPIKeyForProvider:   llmAPIKeyForProvider,
		LLMModelForProvider:    llmModelForProvider,
		RegistryFromViper:      registryFromViper,
		RegisterPlanTool:       registerPlanTool,
		GuardFromViper:         guardFromViper,
		PromptSpecForTelegram: func(ctx context.Context, logger *slog.Logger, logOpts agent.LogOptions, task string, client llm.Client, model string, stickySkills []string) (agent.PromptSpec, []string, []string, error) {
			cfg := skillsConfigFromViper(model)
			if len(stickySkills) > 0 {
				cfg.Requested = append(cfg.Requested, stickySkills...)
			}
			return promptSpecWithSkills(ctx, logger, logOpts, task, client, model, cfg)
		},
		FormatFinalOutput:  formatFinalOutput,
		BuildHeartbeatTask: buildHeartbeatTask,
		BuildHeartbeatMeta: func(source string, interval time.Duration, checklistPath string, checklistEmpty bool, extra map[string]any) map[string]any {
			return buildHeartbeatMeta(source, interval, checklistPath, checklistEmpty, nil, extra)
		},
		BuildHeartbeatProgressSnapshot: func(mgr *memory.Manager, maxItems int) (string, error) {
			return buildHeartbeatProgressSnapshot(mgr, maxItems)
		},
	})
}
