package skillsutil

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/quailyquaily/mistermorph/agent"
)

func TestPromptSpecWithSkills_LoadAllWildcard(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "alpha")
	writeSkill(t, root, "beta")

	spec, loaded, _, err := PromptSpecWithSkills(
		context.Background(),
		nil,
		agent.DefaultLogOptions(),
		"task",
		nil,
		"gpt-5.2",
		SkillsConfig{
			Roots:     []string{root},
			Mode:      "on",
			Requested: []string{"*"},
			Auto:      false,
		},
	)
	if err != nil {
		t.Fatalf("PromptSpecWithSkills: %v", err)
	}
	if len(spec.Blocks) != 2 {
		t.Fatalf("expected 2 loaded skill blocks, got %d", len(spec.Blocks))
	}
	sort.Strings(loaded)
	if len(loaded) != 2 || loaded[0] != "alpha" || loaded[1] != "beta" {
		t.Fatalf("unexpected loaded skills: %#v", loaded)
	}
}

func TestPromptSpecWithSkills_LoadAllWildcardIgnoresUnknownRequests(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "alpha")
	writeSkill(t, root, "beta")

	_, loaded, _, err := PromptSpecWithSkills(
		context.Background(),
		nil,
		agent.DefaultLogOptions(),
		"task",
		nil,
		"gpt-5.2",
		SkillsConfig{
			Roots:     []string{root},
			Mode:      "on",
			Requested: []string{"*", "missing-skill"},
			Auto:      false,
		},
	)
	if err != nil {
		t.Fatalf("PromptSpecWithSkills with wildcard should not fail on unknown skill: %v", err)
	}
	sort.Strings(loaded)
	if len(loaded) != 2 || loaded[0] != "alpha" || loaded[1] != "beta" {
		t.Fatalf("unexpected loaded skills: %#v", loaded)
	}
}

func TestPromptSpecWithSkills_InjectsSkillMetadataOnly(t *testing.T) {
	root := t.TempDir()
	writeSkillWithFrontmatter(t, root, "jsonbill", `---
name: jsonbill
description: Generate invoice PDF.
auth_profiles: ["jsonbill"]
requirements:
  - http_client
  - optional: file_send (chat)
---

# JSONBill

very long instructions that should not be injected
`)

	spec, loaded, _, err := PromptSpecWithSkills(
		context.Background(),
		nil,
		agent.DefaultLogOptions(),
		"task",
		nil,
		"gpt-5.2",
		SkillsConfig{
			Roots:     []string{root},
			Mode:      "on",
			Requested: []string{"jsonbill"},
			Auto:      false,
		},
	)
	if err != nil {
		t.Fatalf("PromptSpecWithSkills: %v", err)
	}
	if len(loaded) != 1 || loaded[0] != "jsonbill" {
		t.Fatalf("unexpected loaded skills: %#v", loaded)
	}
	if len(spec.Blocks) < 1 {
		t.Fatalf("expected at least 1 block, got %d", len(spec.Blocks))
	}
	if len(spec.Blocks) != 1 {
		t.Fatalf("expected only 1 skill metadata block, got %d", len(spec.Blocks))
	}
	content := spec.Blocks[0].Content
	if !strings.Contains(content, "Name: jsonbill") {
		t.Fatalf("skill name missing from prompt block: %q", content)
	}
	if !strings.Contains(content, "Description: Generate invoice PDF.") {
		t.Fatalf("skill description missing from prompt block: %q", content)
	}
	if !strings.Contains(content, "- http_client") || !strings.Contains(content, "- optional: file_send (chat)") {
		t.Fatalf("skill requirements missing from prompt block: %q", content)
	}
	if !strings.Contains(content, "- auth_profile: jsonbill") {
		t.Fatalf("auth_profile requirement missing from prompt block: %q", content)
	}
	if strings.Contains(content, "very long instructions that should not be injected") {
		t.Fatalf("skill full contents should not be injected: %q", content)
	}
}

func writeSkill(t *testing.T, root, id string) {
	t.Helper()
	dir := filepath.Join(root, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir skill dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("# "+id+"\n"), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
}

func writeSkillWithFrontmatter(t *testing.T, root, id, content string) {
	t.Helper()
	dir := filepath.Join(root, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir skill dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
}
