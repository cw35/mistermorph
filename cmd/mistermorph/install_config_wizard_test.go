package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func TestFindReadableInstallConfigPriority(t *testing.T) {
	initViperDefaults()

	root := t.TempDir()
	installDir := filepath.Join(root, "install")
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		t.Fatalf("mkdir install dir: %v", err)
	}

	flagCfgPath := filepath.Join(root, "cfg-from-flag.yaml")
	if err := os.WriteFile(flagCfgPath, []byte("llm:\n  provider: openai\n"), 0o644); err != nil {
		t.Fatalf("write flag config: %v", err)
	}

	dirCfgPath := filepath.Join(installDir, "config.yaml")
	if err := os.WriteFile(dirCfgPath, []byte("llm:\n  provider: gemini\n"), 0o644); err != nil {
		t.Fatalf("write dir config: %v", err)
	}

	home := filepath.Join(root, "home")
	t.Setenv("HOME", home)
	morphHome := filepath.Join(home, ".morph")
	if err := os.MkdirAll(morphHome, 0o755); err != nil {
		t.Fatalf("mkdir ~/.morph: %v", err)
	}
	homeCfgPath := filepath.Join(morphHome, "config.yaml")
	if err := os.WriteFile(homeCfgPath, []byte("llm:\n  provider: cloudflare\n"), 0o644); err != nil {
		t.Fatalf("write ~/.morph/config.yaml: %v", err)
	}

	prevConfig := viper.GetString("config")
	viper.Set("config", flagCfgPath)
	t.Cleanup(func() {
		if prevConfig == "" {
			viper.Set("config", nil)
			return
		}
		viper.Set("config", prevConfig)
	})

	if got, ok := findReadableInstallConfig(nil, installDir); !ok || got != flagCfgPath {
		t.Fatalf("findReadableInstallConfig() = (%q, %v), want (%q, true)", got, ok, flagCfgPath)
	}

	viper.Set("config", "")
	if got, ok := findReadableInstallConfig(nil, installDir); !ok || got != dirCfgPath {
		t.Fatalf("findReadableInstallConfig() = (%q, %v), want (%q, true)", got, ok, dirCfgPath)
	}

	if err := os.Remove(dirCfgPath); err != nil {
		t.Fatalf("remove dir config: %v", err)
	}
	if got, ok := findReadableInstallConfig(nil, installDir); !ok || got != homeCfgPath {
		t.Fatalf("findReadableInstallConfig() = (%q, %v), want (%q, true)", got, ok, homeCfgPath)
	}
}

func TestMaybeCollectInstallConfigSetup_NonInteractiveSkipsWizard(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetIn(bytes.NewBufferString(""))
	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)

	setup, err := maybeCollectInstallConfigSetup(cmd, false)
	if err != nil {
		t.Fatalf("maybeCollectInstallConfigSetup() error = %v", err)
	}
	if setup != nil {
		t.Fatalf("expected nil setup in non-interactive mode")
	}
	if !strings.Contains(errOut.String(), "non-interactive mode detected") {
		t.Fatalf("expected warning about non-interactive mode, got: %q", errOut.String())
	}
}

func TestPatchInitConfigWithSetup_AppliesOverrides(t *testing.T) {
	body, err := loadConfigExample()
	if err != nil {
		t.Fatalf("loadConfigExample() error = %v", err)
	}

	setup := &installConfigSetup{
		Provider:                 "cloudflare",
		Endpoint:                 "https://api.cloudflare.com/client/v4",
		Model:                    "@cf/meta/llama-3.1-8b-instruct",
		CloudflareAccount:        "acc-123",
		CloudflareAPIToken:       "token-xyz",
		TelegramBotToken:         "tg-token",
		TelegramGroupTriggerMode: "smart",
		ConfigureSlack:           true,
		SlackBotToken:            "xoxb-test",
		SlackAppToken:            "xapp-test",
		SlackGroupTrigger:        "talkative",
	}

	got := patchInitConfigWithSetup(body, "/tmp/my-state", setup)

	assertContains := func(substr string) {
		t.Helper()
		if !strings.Contains(got, substr) {
			t.Fatalf("patched config missing %q", substr)
		}
	}

	assertContains(`file_state_dir: "/tmp/my-state"`)
	assertContains(`provider: cloudflare`)
	assertContains(`endpoint: "https://api.cloudflare.com/client/v4"`)
	assertContains(`model: "@cf/meta/llama-3.1-8b-instruct"`)
	assertContains(`api_key: "" # or set via MISTER_MORPH_LLM_API_KEY`)
	assertContains(`account_id: "acc-123"`)
	assertContains(`api_token: "token-xyz"`)
	assertContains(`bot_token: "tg-token"`)
	assertContains(`group_trigger_mode: "smart"`)
	assertContains(`bot_token: "xoxb-test"`)
	assertContains(`app_token: "xapp-test"`)
	assertContains(`group_trigger_mode: "talkative"`)
}
