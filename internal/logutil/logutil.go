package logutil

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/quailyquaily/mistermorph/agent"
	"github.com/spf13/viper"
)

type ConfigReader interface {
	GetString(string) string
	GetBool(string) bool
	GetInt(string) int
	GetStringSlice(string) []string
	IsSet(string) bool
}

type LoggerConfig struct {
	Level     string
	Format    string
	AddSource bool
}

type LogOptionsConfig struct {
	IncludeThoughts      bool
	IncludeToolParams    bool
	IncludeSkillContents bool
	MaxThoughtChars      int
	MaxJSONBytes         int
	MaxStringValueChars  int
	MaxSkillContentChars int
	RedactKeys           []string
	RedactKeysSet        bool
}

func LoggerConfigFromReader(r ConfigReader) LoggerConfig {
	if r == nil {
		return LoggerConfig{}
	}
	return LoggerConfig{
		Level:     r.GetString("logging.level"),
		Format:    r.GetString("logging.format"),
		AddSource: r.GetBool("logging.add_source"),
	}
}

func LoggerConfigFromViper() LoggerConfig {
	return LoggerConfigFromReader(viper.GetViper())
}

func LoggerFromConfig(cfg LoggerConfig) (*slog.Logger, error) {
	return newLoggerFromConfig(cfg)
}

func LoggerFromViper() (*slog.Logger, error) {
	return LoggerFromConfig(LoggerConfigFromViper())
}

func LogOptionsConfigFromReader(r ConfigReader) LogOptionsConfig {
	if r == nil {
		return LogOptionsConfig{}
	}
	return LogOptionsConfig{
		IncludeThoughts:      r.GetBool("logging.include_thoughts"),
		IncludeToolParams:    r.GetBool("logging.include_tool_params"),
		IncludeSkillContents: r.GetBool("logging.include_skill_contents"),
		MaxThoughtChars:      r.GetInt("logging.max_thought_chars"),
		MaxJSONBytes:         r.GetInt("logging.max_json_bytes"),
		MaxStringValueChars:  r.GetInt("logging.max_string_value_chars"),
		MaxSkillContentChars: r.GetInt("logging.max_skill_content_chars"),
		RedactKeys:           append([]string(nil), r.GetStringSlice("logging.redact_keys")...),
		RedactKeysSet:        r.IsSet("logging.redact_keys"),
	}
}

func LogOptionsConfigFromViper() LogOptionsConfig {
	return LogOptionsConfigFromReader(viper.GetViper())
}

func LogOptionsFromConfig(cfg LogOptionsConfig) agent.LogOptions {
	logOpts := agent.DefaultLogOptions()
	logOpts.IncludeThoughts = cfg.IncludeThoughts
	logOpts.IncludeToolParams = cfg.IncludeToolParams
	logOpts.IncludeSkillContents = cfg.IncludeSkillContents
	logOpts.MaxThoughtChars = cfg.MaxThoughtChars
	logOpts.MaxJSONBytes = cfg.MaxJSONBytes
	logOpts.MaxStringValueChars = cfg.MaxStringValueChars
	logOpts.MaxSkillContentChars = cfg.MaxSkillContentChars
	if cfg.RedactKeysSet && len(cfg.RedactKeys) > 0 {
		logOpts.RedactKeys = append([]string(nil), cfg.RedactKeys...)
	}
	return logOpts
}

func LogOptionsFromViper() agent.LogOptions {
	return LogOptionsFromConfig(LogOptionsConfigFromViper())
}

func newLoggerFromConfig(cfg LoggerConfig) (*slog.Logger, error) {
	level, err := parseSlogLevel(cfg.Level)
	if err != nil {
		return nil, err
	}

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: cfg.AddSource,
	}

	var h slog.Handler
	switch strings.ToLower(strings.TrimSpace(cfg.Format)) {
	case "", "text":
		h = slog.NewTextHandler(os.Stderr, opts)
	case "json":
		h = slog.NewJSONHandler(os.Stderr, opts)
	default:
		return nil, fmt.Errorf("unknown logging.format: %s", cfg.Format)
	}

	return slog.New(h), nil
}

func parseSlogLevel(s string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "info":
		return slog.LevelInfo, nil
	case "debug":
		return slog.LevelDebug, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("unknown logging.level: %s", s)
	}
}
