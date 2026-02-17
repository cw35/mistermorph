package main

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/quailyquaily/mistermorph/guard"
	"github.com/quailyquaily/mistermorph/internal/pathutil"
	"github.com/spf13/viper"
)

func guardFromViper(log *slog.Logger) *guard.Guard {
	return buildGuardFromConfig(loadGuardConfigFromViper(), log)
}

type guardConfigSnapshot struct {
	Enabled bool
	Config  guard.Config
	Dir     string
}

func loadGuardConfigFromViper() guardConfigSnapshot {
	var patterns []guard.RegexPattern
	_ = viper.UnmarshalKey("guard.redaction.patterns", &patterns)

	return guardConfigSnapshot{
		Enabled: viper.GetBool("guard.enabled"),
		Config: guard.Config{
			Enabled: true,
			Network: guard.NetworkConfig{
				URLFetch: guard.URLFetchNetworkPolicy{
					AllowedURLPrefixes: append([]string(nil), viper.GetStringSlice("guard.network.url_fetch.allowed_url_prefixes")...),
					DenyPrivateIPs:     viper.GetBool("guard.network.url_fetch.deny_private_ips"),
					FollowRedirects:    viper.GetBool("guard.network.url_fetch.follow_redirects"),
					AllowProxy:         viper.GetBool("guard.network.url_fetch.allow_proxy"),
				},
			},
			Redaction: guard.RedactionConfig{
				Enabled:  viper.GetBool("guard.redaction.enabled"),
				Patterns: append([]guard.RegexPattern(nil), patterns...),
			},
			Audit: guard.AuditConfig{
				JSONLPath:      strings.TrimSpace(viper.GetString("guard.audit.jsonl_path")),
				RotateMaxBytes: viper.GetInt64("guard.audit.rotate_max_bytes"),
			},
			Approvals: guard.ApprovalsConfig{
				Enabled: viper.GetBool("guard.approvals.enabled"),
			},
		},
		Dir: resolveGuardDirFromValues(viper.GetString("file_state_dir"), viper.GetString("guard.dir_name")),
	}
}

func buildGuardFromConfig(cfg guardConfigSnapshot, log *slog.Logger) *guard.Guard {
	if !cfg.Enabled {
		return nil
	}
	if log == nil {
		log = slog.Default()
	}

	guardDir := strings.TrimSpace(cfg.Dir)
	if guardDir == "" {
		guardDir = resolveGuardDirFromValues("", "")
	}
	if err := os.MkdirAll(guardDir, 0o700); err != nil {
		log.Warn("guard_dir_create_error", "error", err.Error(), "guard_dir", guardDir)
		return nil
	}
	lockRoot := filepath.Join(guardDir, ".fslocks")

	jsonlPath := strings.TrimSpace(cfg.Config.Audit.JSONLPath)
	if jsonlPath == "" {
		jsonlPath = filepath.Join(guardDir, "audit", "guard_audit.jsonl")
	}
	jsonlPath = pathutil.ExpandHomePath(jsonlPath)

	var sink guard.AuditSink
	var warnings []string
	if strings.TrimSpace(jsonlPath) != "" {
		s, err := guard.NewJSONLAuditSink(jsonlPath, cfg.Config.Audit.RotateMaxBytes, lockRoot)
		if err != nil {
			log.Warn("guard_audit_sink_error", "error", err.Error())
			warnings = append(warnings, "guard_audit_sink_error: "+err.Error())
		} else {
			sink = s
		}
	}

	var approvals guard.ApprovalStore
	if cfg.Config.Approvals.Enabled {
		approvalsPath := filepath.Join(guardDir, "approvals", "guard_approvals.json")
		st, err := guard.NewFileApprovalStore(approvalsPath, lockRoot)
		if err != nil {
			log.Warn("guard_approvals_store_error", "error", err.Error())
			warnings = append(warnings, "guard_approvals_store_error: "+err.Error())
		} else {
			approvals = st
		}
	}

	log.Info("guard_enabled",
		"guard_dir", guardDir,
		"url_fetch_prefixes", len(cfg.Config.Network.URLFetch.AllowedURLPrefixes),
		"audit_jsonl", jsonlPath,
		"approvals_enabled", approvals != nil,
	)

	if len(warnings) > 0 {
		return guard.NewWithWarnings(cfg.Config, sink, approvals, warnings)
	}
	return guard.New(cfg.Config, sink, approvals)
}

func resolveGuardDirFromValues(fileStateDir, guardDirName string) string {
	base := pathutil.ResolveStateDir(fileStateDir)
	home, err := os.UserHomeDir()
	if strings.TrimSpace(base) == "" && err == nil && strings.TrimSpace(home) != "" {
		base = filepath.Join(home, ".morph")
	}
	if strings.TrimSpace(base) == "" {
		base = filepath.Join(".", ".morph")
	}
	name := strings.TrimSpace(guardDirName)
	if name == "" {
		name = "guard"
	}
	return filepath.Join(base, name)
}
