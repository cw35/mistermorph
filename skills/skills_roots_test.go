package skills

import (
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestDefaultRoots_UsesFileStateDirSkills(t *testing.T) {
	prevStateDir := viper.GetString("file_state_dir")
	prevSkillsDirName := viper.GetString("skills.dir_name")
	t.Cleanup(func() {
		viper.Set("file_state_dir", prevStateDir)
		viper.Set("skills.dir_name", prevSkillsDirName)
	})

	stateDir := filepath.Join(t.TempDir(), "state")
	viper.Set("file_state_dir", stateDir)
	viper.Set("skills.dir_name", "skills")

	got := DefaultRoots()
	if len(got) != 1 {
		t.Fatalf("DefaultRoots() count = %d, want 1", len(got))
	}
	want := filepath.Clean(filepath.Join(stateDir, "skills"))
	if filepath.Clean(got[0]) != want {
		t.Fatalf("DefaultRoots()[0] = %q, want %q", got[0], want)
	}
}

func TestNormalizeRoots_DedupesAndPrioritizesDefaultSkillsRoot(t *testing.T) {
	prevStateDir := viper.GetString("file_state_dir")
	prevSkillsDirName := viper.GetString("skills.dir_name")
	t.Cleanup(func() {
		viper.Set("file_state_dir", prevStateDir)
		viper.Set("skills.dir_name", prevSkillsDirName)
	})

	stateDir := filepath.Join(t.TempDir(), "state")
	viper.Set("file_state_dir", stateDir)
	viper.Set("skills.dir_name", "my_skills")

	defaultSkillsRoot := filepath.Join(stateDir, "my_skills")
	custom := filepath.Join(t.TempDir(), "custom-skills")

	got := normalizeRoots([]string{
		custom,
		defaultSkillsRoot,
		custom,
	})

	if len(got) != 2 {
		t.Fatalf("normalizeRoots() count = %d, want 2: %#v", len(got), got)
	}
	if filepath.Clean(got[0]) != filepath.Clean(defaultSkillsRoot) {
		t.Fatalf("normalizeRoots()[0] = %q, want %q", got[0], defaultSkillsRoot)
	}
	if filepath.Clean(got[1]) != filepath.Clean(custom) {
		t.Fatalf("normalizeRoots()[1] = %q, want %q", got[1], custom)
	}
}
