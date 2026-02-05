package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/quailyquaily/mistermorph/assets"
	"github.com/quailyquaily/mistermorph/internal/pathutil"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init [dir]",
		Short: "Initialize config.yaml, HEARTBEAT.md, and install built-in skills",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "~/.morph/"
			if len(args) == 1 && strings.TrimSpace(args[0]) != "" {
				dir = args[0]
			}
			dir = pathutil.ExpandHomePath(dir)
			if strings.TrimSpace(dir) == "" {
				return fmt.Errorf("invalid dir")
			}
			dir = filepath.Clean(dir)

			if err := os.MkdirAll(dir, 0o755); err != nil {
				return err
			}

			cfgPath := filepath.Join(dir, "config.yaml")
			writeConfig := true
			if _, err := os.Stat(cfgPath); err == nil {
				fmt.Fprintf(os.Stderr, "warn: config already exists, skipping: %s\n", cfgPath)
				writeConfig = false
			}

			hbPath := filepath.Join(dir, "HEARTBEAT.md")
			writeHeartbeat := true
			if _, err := os.Stat(hbPath); err == nil {
				fmt.Fprintf(os.Stderr, "warn: heartbeat already exists, skipping: %s\n", hbPath)
				writeHeartbeat = false
			}

			if writeConfig {
				cfgBody, err := loadConfigExample()
				if err != nil {
					return err
				}
				cfgBody = patchInitConfig(cfgBody, dir)

				if err := os.WriteFile(cfgPath, []byte(cfgBody), 0o644); err != nil {
					return err
				}
			}

			if writeHeartbeat {
				hbBody, err := loadHeartbeatTemplate()
				if err != nil {
					return err
				}
				if err := os.WriteFile(hbPath, []byte(hbBody), 0o644); err != nil {
					return err
				}
			}

			skillsDir := filepath.Join(dir, "skills")
			if err := installBuiltInSkills(skillsDir, false, false, false); err != nil {
				return err
			}

			fmt.Printf("initialized %s\n", dir)
			return nil
		},
	}

	return cmd
}

func loadConfigExample() (string, error) {
	data, err := assets.ConfigFS.ReadFile("config/config.example.yaml")
	if err != nil {
		return "", fmt.Errorf("read embedded config.example.yaml: %w", err)
	}
	return string(data), nil
}

func loadHeartbeatTemplate() (string, error) {
	data, err := assets.ConfigFS.ReadFile("config/HEARTBEAT.md")
	if err != nil {
		return "", fmt.Errorf("read embedded HEARTBEAT.md: %w", err)
	}
	return string(data), nil
}

func patchInitConfig(cfg string, dir string) string {
	if strings.TrimSpace(cfg) == "" {
		return cfg
	}
	dir = filepath.Clean(dir)
	dir = filepath.ToSlash(dir)
	cfg = strings.ReplaceAll(cfg, `file_state_dir: "~/.morph"`, fmt.Sprintf(`file_state_dir: "%s"`, dir))
	return cfg
}
