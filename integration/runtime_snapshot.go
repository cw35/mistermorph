package integration

import (
	"log/slog"
	"strings"
	"time"

	"github.com/quailyquaily/mistermorph/agent"
	"github.com/quailyquaily/mistermorph/guard"
	"github.com/quailyquaily/mistermorph/internal/channelopts"
	"github.com/quailyquaily/mistermorph/internal/llmutil"
	"github.com/quailyquaily/mistermorph/internal/logutil"
	"github.com/quailyquaily/mistermorph/internal/skillsutil"
	"github.com/quailyquaily/mistermorph/secrets"
)

type runtimeSnapshot struct {
	Logger                      *slog.Logger
	LoggerInitErr               error
	LogOptions                  agent.LogOptions
	LLMValues                   llmutil.RuntimeValues
	LLMProvider                 string
	LLMEndpoint                 string
	LLMAPIKey                   string
	LLMModel                    string
	LLMRequestTimeout           time.Duration
	AgentMaxSteps               int
	AgentParseRetries           int
	AgentMaxTokenBudget         int
	SecretsRequireSkillProfiles bool
	SkillsConfig                skillsutil.SkillsConfig
	Registry                    registrySnapshot
	Guard                       guardSnapshot
	Telegram                    channelopts.TelegramConfig
	Slack                       channelopts.SlackConfig
}

type runtimeSnapshotInput struct {
	LoggerConfig                logutil.LoggerConfig
	LogOptionsConfig            logutil.LogOptionsConfig
	LLMValues                   llmutil.RuntimeValues
	LLMRequestTimeout           time.Duration
	AgentMaxSteps               int
	AgentParseRetries           int
	AgentMaxTokenBudget         int
	SecretsRequireSkillProfiles bool
	SkillsConfig                skillsutil.SkillsConfig
	Registry                    registrySnapshot
	Guard                       guardSnapshot
	Telegram                    channelopts.TelegramConfig
	Slack                       channelopts.SlackConfig
}

type registrySnapshot struct {
	UserAgent                     string
	SecretsEnabled                bool
	SecretsRequireSkillProfiles   bool
	SecretsAllowProfiles          []string
	SecretsAliases                map[string]string
	AuthProfiles                  map[string]secrets.AuthProfile
	FileCacheDir                  string
	FileStateDir                  string
	ToolsReadFileMaxBytes         int64
	ToolsReadFileDenyPaths        []string
	ToolsWriteFileEnabled         bool
	ToolsWriteFileMaxBytes        int
	ToolsBashEnabled              bool
	ToolsBashTimeout              time.Duration
	ToolsBashMaxOutputBytes       int
	ToolsBashDenyPaths            []string
	ToolsURLFetchEnabled          bool
	ToolsURLFetchTimeout          time.Duration
	ToolsURLFetchMaxBytes         int64
	ToolsURLFetchMaxBytesDownload int64
	ToolsWebSearchEnabled         bool
	ToolsWebSearchTimeout         time.Duration
	ToolsWebSearchMaxResults      int
	ToolsWebSearchBaseURL         string
	ToolsContactsEnabled          bool
	ToolsTodoUpdateEnabled        bool
	TODOPathWIP                   string
	TODOPathDone                  string
	ContactsDir                   string
	MAEPDir                       string
	TelegramBotToken              string
	TelegramBaseURL               string
	SlackBotToken                 string
	SlackBaseURL                  string
	ContactsFailureCooldown       time.Duration
}

type guardSnapshot struct {
	Enabled bool
	Config  guard.Config
	Dir     string
}

func newRuntimeSnapshot(in runtimeSnapshotInput) runtimeSnapshot {
	logger, err := logutil.LoggerFromConfig(in.LoggerConfig)
	if err != nil {
		logger = slog.Default()
	}
	logOpts := cloneLogOptions(logutil.LogOptionsFromConfig(in.LogOptionsConfig))
	values := in.LLMValues
	provider := strings.TrimSpace(values.Provider)

	return runtimeSnapshot{
		Logger:                      logger,
		LoggerInitErr:               err,
		LogOptions:                  logOpts,
		LLMValues:                   values,
		LLMProvider:                 provider,
		LLMEndpoint:                 llmutil.EndpointForProviderWithValues(provider, values),
		LLMAPIKey:                   llmutil.APIKeyForProviderWithValues(provider, values),
		LLMModel:                    llmutil.ModelForProviderWithValues(provider, values),
		LLMRequestTimeout:           in.LLMRequestTimeout,
		AgentMaxSteps:               in.AgentMaxSteps,
		AgentParseRetries:           in.AgentParseRetries,
		AgentMaxTokenBudget:         in.AgentMaxTokenBudget,
		SecretsRequireSkillProfiles: in.SecretsRequireSkillProfiles,
		SkillsConfig:                cloneSkillsConfig(in.SkillsConfig),
		Registry:                    cloneRegistrySnapshot(in.Registry),
		Guard:                       cloneGuardSnapshot(in.Guard),
		Telegram:                    cloneTelegramConfig(in.Telegram),
		Slack:                       cloneSlackConfig(in.Slack),
	}
}

func cloneLogOptions(in agent.LogOptions) agent.LogOptions {
	out := in
	out.RedactKeys = append([]string(nil), in.RedactKeys...)
	return out
}

func cloneSkillsConfig(in skillsutil.SkillsConfig) skillsutil.SkillsConfig {
	out := in
	out.Roots = append([]string(nil), in.Roots...)
	out.Requested = append([]string(nil), in.Requested...)
	return out
}

func cloneRegistrySnapshot(in registrySnapshot) registrySnapshot {
	out := in
	out.SecretsAllowProfiles = append([]string(nil), in.SecretsAllowProfiles...)
	out.SecretsAliases = copyStringMap(in.SecretsAliases)
	out.AuthProfiles = copyAuthProfilesMap(in.AuthProfiles)
	out.ToolsReadFileDenyPaths = append([]string(nil), in.ToolsReadFileDenyPaths...)
	out.ToolsBashDenyPaths = append([]string(nil), in.ToolsBashDenyPaths...)
	return out
}

func cloneGuardSnapshot(in guardSnapshot) guardSnapshot {
	out := in
	out.Config.Network.URLFetch.AllowedURLPrefixes = append([]string(nil), in.Config.Network.URLFetch.AllowedURLPrefixes...)
	out.Config.Redaction.Patterns = append([]guard.RegexPattern(nil), in.Config.Redaction.Patterns...)
	return out
}

func cloneTelegramConfig(in channelopts.TelegramConfig) channelopts.TelegramConfig {
	out := in
	out.AllowedChatIDsRaw = append([]string(nil), in.AllowedChatIDsRaw...)
	out.MAEPListenAddrs = append([]string(nil), in.MAEPListenAddrs...)
	return out
}

func cloneSlackConfig(in channelopts.SlackConfig) channelopts.SlackConfig {
	out := in
	out.AllowedTeamIDs = append([]string(nil), in.AllowedTeamIDs...)
	out.AllowedChannelIDs = append([]string(nil), in.AllowedChannelIDs...)
	return out
}

func copyStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func copyAuthProfilesMap(in map[string]secrets.AuthProfile) map[string]secrets.AuthProfile {
	if len(in) == 0 {
		return map[string]secrets.AuthProfile{}
	}
	out := make(map[string]secrets.AuthProfile, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
