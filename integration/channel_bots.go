package integration

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/quailyquaily/mistermorph/agent"
	"github.com/quailyquaily/mistermorph/guard"
	"github.com/quailyquaily/mistermorph/internal/channelopts"
	slackruntime "github.com/quailyquaily/mistermorph/internal/channelruntime/slack"
	telegramruntime "github.com/quailyquaily/mistermorph/internal/channelruntime/telegram"
	"github.com/quailyquaily/mistermorph/internal/heartbeatutil"
	"github.com/quailyquaily/mistermorph/internal/llmconfig"
	"github.com/quailyquaily/mistermorph/internal/llmutil"
	"github.com/quailyquaily/mistermorph/internal/skillsutil"
	"github.com/quailyquaily/mistermorph/internal/toolsutil"
	"github.com/quailyquaily/mistermorph/llm"
	"github.com/quailyquaily/mistermorph/tools"
)

// BotRunner controls a long-running channel bot lifecycle.
type BotRunner interface {
	Run(ctx context.Context) error
	Close() error
}

type TelegramOptions struct {
	BotToken                      string
	AllowedChatIDs                []int64
	PollTimeout                   time.Duration
	TaskTimeout                   time.Duration
	MaxConcurrency                int
	GroupTriggerMode              string
	AddressingConfidenceThreshold float64
	AddressingInterjectThreshold  float64
	Hooks                         TelegramHooks
}

type SlackOptions struct {
	BotToken                      string
	AppToken                      string
	AllowedTeamIDs                []string
	AllowedChannelIDs             []string
	TaskTimeout                   time.Duration
	MaxConcurrency                int
	GroupTriggerMode              string
	AddressingConfidenceThreshold float64
	AddressingInterjectThreshold  float64
	Hooks                         SlackHooks
}

type TelegramHooks struct {
	OnInbound  func(TelegramInboundEvent)
	OnOutbound func(TelegramOutboundEvent)
	OnError    func(TelegramErrorEvent)
}

type TelegramInboundEvent = telegramruntime.InboundEvent
type TelegramOutboundEvent = telegramruntime.OutboundEvent
type TelegramErrorEvent = telegramruntime.ErrorEvent

type SlackHooks struct {
	OnInbound  func(SlackInboundEvent)
	OnOutbound func(SlackOutboundEvent)
	OnError    func(SlackErrorEvent)
}

type SlackInboundEvent = slackruntime.InboundEvent
type SlackOutboundEvent = slackruntime.OutboundEvent
type SlackErrorEvent = slackruntime.ErrorEvent

func (rt *Runtime) NewTelegramBot(opts TelegramOptions) (BotRunner, error) {
	if rt == nil {
		return nil, fmt.Errorf("runtime is nil")
	}
	if strings.TrimSpace(opts.BotToken) == "" {
		return nil, fmt.Errorf("telegram bot token is required")
	}
	return &telegramBotRunner{rt: rt, opts: opts}, nil
}

func (rt *Runtime) NewSlackBot(opts SlackOptions) (BotRunner, error) {
	if rt == nil {
		return nil, fmt.Errorf("runtime is nil")
	}
	if strings.TrimSpace(opts.BotToken) == "" {
		return nil, fmt.Errorf("slack bot token is required")
	}
	if strings.TrimSpace(opts.AppToken) == "" {
		return nil, fmt.Errorf("slack app token is required")
	}
	return &slackBotRunner{rt: rt, opts: opts}, nil
}

type telegramBotRunner struct {
	rt    *Runtime
	opts  TelegramOptions
	state runState
}

func (r *telegramBotRunner) Run(ctx context.Context) error {
	if r == nil || r.rt == nil {
		return fmt.Errorf("telegram runner is nil")
	}
	runCtx, cancel, err := r.state.begin(ctx, "telegram")
	if err != nil {
		return err
	}
	defer r.state.end(cancel)

	snap, err := r.rt.snapshot()
	if err != nil {
		return err
	}

	runOpts, err := channelopts.BuildTelegramRunOptions(snap.Telegram, channelopts.TelegramInput{
		BotToken:                      strings.TrimSpace(r.opts.BotToken),
		AllowedChatIDs:                append([]int64(nil), r.opts.AllowedChatIDs...),
		GroupTriggerMode:              strings.TrimSpace(r.opts.GroupTriggerMode),
		AddressingConfidenceThreshold: r.opts.AddressingConfidenceThreshold,
		AddressingInterjectThreshold:  r.opts.AddressingInterjectThreshold,
		WithMAEP:                      false,
		PollTimeout:                   r.opts.PollTimeout,
		TaskTimeout:                   r.opts.TaskTimeout,
		MaxConcurrency:                r.opts.MaxConcurrency,
		Hooks:                         r.runtimeHooks(),
		InspectPrompt:                 r.rt.cfg.Inspect.Prompt,
		InspectRequest:                r.rt.cfg.Inspect.Request,
	})
	if err != nil {
		return err
	}
	return telegramruntime.Run(runCtx, r.rt.telegramDependencies(snap), runOpts)
}

func (r *telegramBotRunner) Close() error {
	if r == nil {
		return nil
	}
	return r.state.close()
}

type slackBotRunner struct {
	rt    *Runtime
	opts  SlackOptions
	state runState
}

func (r *slackBotRunner) Run(ctx context.Context) error {
	if r == nil || r.rt == nil {
		return fmt.Errorf("slack runner is nil")
	}
	runCtx, cancel, err := r.state.begin(ctx, "slack")
	if err != nil {
		return err
	}
	defer r.state.end(cancel)

	snap, err := r.rt.snapshot()
	if err != nil {
		return err
	}

	runOpts := channelopts.BuildSlackRunOptions(snap.Slack, channelopts.SlackInput{
		BotToken:                      strings.TrimSpace(r.opts.BotToken),
		AppToken:                      strings.TrimSpace(r.opts.AppToken),
		AllowedTeamIDs:                append([]string(nil), r.opts.AllowedTeamIDs...),
		AllowedChannelIDs:             append([]string(nil), r.opts.AllowedChannelIDs...),
		GroupTriggerMode:              strings.TrimSpace(r.opts.GroupTriggerMode),
		AddressingConfidenceThreshold: r.opts.AddressingConfidenceThreshold,
		AddressingInterjectThreshold:  r.opts.AddressingInterjectThreshold,
		TaskTimeout:                   r.opts.TaskTimeout,
		MaxConcurrency:                r.opts.MaxConcurrency,
		Hooks:                         r.runtimeHooks(),
		InspectPrompt:                 r.rt.cfg.Inspect.Prompt,
		InspectRequest:                r.rt.cfg.Inspect.Request,
	})
	return slackruntime.Run(runCtx, r.rt.slackDependencies(snap), runOpts)
}

func (r *slackBotRunner) Close() error {
	if r == nil {
		return nil
	}
	return r.state.close()
}

type runState struct {
	mu      sync.Mutex
	running bool
	cancel  context.CancelFunc
}

func (s *runState) begin(ctx context.Context, name string) (context.Context, context.CancelFunc, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	runCtx, cancel := context.WithCancel(ctx)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		cancel()
		return nil, nil, fmt.Errorf("%s runner already running", strings.TrimSpace(name))
	}
	s.running = true
	s.cancel = cancel
	return runCtx, cancel, nil
}

func (s *runState) end(cancel context.CancelFunc) {
	s.mu.Lock()
	s.cancel = nil
	s.running = false
	s.mu.Unlock()
	cancel()
}

func (s *runState) close() error {
	s.mu.Lock()
	cancel := s.cancel
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	return nil
}

func (r *telegramBotRunner) runtimeHooks() telegramruntime.Hooks {
	h := r.opts.Hooks
	if h.OnInbound == nil && h.OnOutbound == nil && h.OnError == nil {
		return telegramruntime.Hooks{}
	}
	return telegramruntime.Hooks{
		OnInbound: func(_ context.Context, event telegramruntime.InboundEvent) {
			if h.OnInbound == nil {
				return
			}
			h.OnInbound(event)
		},
		OnOutbound: func(_ context.Context, event telegramruntime.OutboundEvent) {
			if h.OnOutbound == nil {
				return
			}
			h.OnOutbound(event)
		},
		OnError: func(_ context.Context, event telegramruntime.ErrorEvent) {
			if h.OnError == nil {
				return
			}
			h.OnError(event)
		},
	}
}

func (r *slackBotRunner) runtimeHooks() slackruntime.Hooks {
	h := r.opts.Hooks
	if h.OnInbound == nil && h.OnOutbound == nil && h.OnError == nil {
		return slackruntime.Hooks{}
	}
	return slackruntime.Hooks{
		OnInbound: func(_ context.Context, event slackruntime.InboundEvent) {
			if h.OnInbound == nil {
				return
			}
			h.OnInbound(event)
		},
		OnOutbound: func(_ context.Context, event slackruntime.OutboundEvent) {
			if h.OnOutbound == nil {
				return
			}
			h.OnOutbound(event)
		},
		OnError: func(_ context.Context, event slackruntime.ErrorEvent) {
			if h.OnError == nil {
				return
			}
			h.OnError(event)
		},
	}
}

func (rt *Runtime) telegramDependencies(snap runtimeSnapshot) telegramruntime.Dependencies {
	return telegramruntime.Dependencies{
		Logger: func() (*slog.Logger, error) {
			if snap.Logger != nil {
				return snap.Logger, nil
			}
			return slog.Default(), nil
		},
		LogOptions: func() agent.LogOptions {
			return cloneLogOptions(snap.LogOptions)
		},
		CreateLLMClient: func(provider, endpoint, apiKey, model string, timeout time.Duration) (llm.Client, error) {
			return llmutil.ClientFromConfigWithValues(llmconfig.ClientConfig{
				Provider:       provider,
				Endpoint:       endpoint,
				APIKey:         apiKey,
				Model:          model,
				RequestTimeout: timeout,
			}, snap.LLMValues)
		},
		LLMProvider:            func() string { return snap.LLMProvider },
		LLMEndpointForProvider: func(_ string) string { return snap.LLMEndpoint },
		LLMAPIKeyForProvider:   func(_ string) string { return snap.LLMAPIKey },
		LLMModelForProvider:    func(_ string) string { return snap.LLMModel },
		Registry:               func() *tools.Registry { return rt.buildRegistry(snap.Registry, snap.Logger) },
		RegisterPlanTool:       rt.maybeRegisterPlanTool,
		Guard: func(logger *slog.Logger) *guard.Guard {
			return rt.buildGuard(snap.Guard, logger)
		},
		PromptSpec: func(ctx context.Context, logger *slog.Logger, logOpts agent.LogOptions, task string, client llm.Client, model string, stickySkills []string) (agent.PromptSpec, []string, []string, error) {
			return rt.promptSpecWithSkillsFromConfig(ctx, logger, logOpts, task, client, model, snap.SkillsConfig, stickySkills)
		},
		BuildHeartbeatTask: heartbeatutil.BuildHeartbeatTask,
		BuildHeartbeatMeta: func(source string, interval time.Duration, checklistPath string, checklistEmpty bool, extra map[string]any) map[string]any {
			return heartbeatutil.BuildHeartbeatMeta(source, interval, checklistPath, checklistEmpty, nil, extra)
		},
	}
}

func (rt *Runtime) slackDependencies(snap runtimeSnapshot) slackruntime.Dependencies {
	return slackruntime.Dependencies{
		Logger: func() (*slog.Logger, error) {
			if snap.Logger != nil {
				return snap.Logger, nil
			}
			return slog.Default(), nil
		},
		LogOptions: func() agent.LogOptions {
			return cloneLogOptions(snap.LogOptions)
		},
		CreateLLMClient: func(provider, endpoint, apiKey, model string, timeout time.Duration) (llm.Client, error) {
			return llmutil.ClientFromConfigWithValues(llmconfig.ClientConfig{
				Provider:       provider,
				Endpoint:       endpoint,
				APIKey:         apiKey,
				Model:          model,
				RequestTimeout: timeout,
			}, snap.LLMValues)
		},
		LLMProvider:            func() string { return snap.LLMProvider },
		LLMEndpointForProvider: func(_ string) string { return snap.LLMEndpoint },
		LLMAPIKeyForProvider:   func(_ string) string { return snap.LLMAPIKey },
		LLMModelForProvider:    func(_ string) string { return snap.LLMModel },
		Registry:               func() *tools.Registry { return rt.buildRegistry(snap.Registry, snap.Logger) },
		RegisterPlanTool:       rt.maybeRegisterPlanTool,
		Guard: func(logger *slog.Logger) *guard.Guard {
			return rt.buildGuard(snap.Guard, logger)
		},
		PromptSpec: func(ctx context.Context, logger *slog.Logger, logOpts agent.LogOptions, task string, client llm.Client, model string, stickySkills []string) (agent.PromptSpec, []string, []string, error) {
			return rt.promptSpecWithSkillsFromConfig(ctx, logger, logOpts, task, client, model, snap.SkillsConfig, stickySkills)
		},
	}
}

func (rt *Runtime) maybeRegisterPlanTool(reg *tools.Registry, client llm.Client, model string) {
	if rt == nil || !rt.cfg.Features.PlanTool {
		return
	}
	toolsutil.RegisterPlanTool(reg, client, model)
}

func (rt *Runtime) promptSpecWithSkillsFromConfig(ctx context.Context, logger *slog.Logger, logOpts agent.LogOptions, task string, client llm.Client, model string, base skillsutil.SkillsConfig, stickySkills []string) (agent.PromptSpec, []string, []string, error) {
	if rt == nil {
		return agent.PromptSpec{}, nil, nil, fmt.Errorf("runtime is nil")
	}
	if !rt.cfg.Features.Skills {
		return agent.DefaultPromptSpec(), nil, nil, nil
	}
	cfg := cloneSkillsConfig(base)
	if len(stickySkills) > 0 {
		cfg.Requested = append(cfg.Requested, stickySkills...)
	}
	return skillsutil.PromptSpecWithSkills(ctx, logger, logOpts, task, client, model, cfg)
}
