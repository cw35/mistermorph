package skillscmd

import (
	"fmt"

	"github.com/quailyquaily/mistermorph/internal/clifmt"
	"github.com/quailyquaily/mistermorph/internal/statepaths"
	"github.com/quailyquaily/mistermorph/skills"
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skills",
		Short: "Discover and install SKILL.md skills",
	}

	cmd.AddCommand(newSkillsListCmd())
	cmd.AddCommand(NewSkillsInstallBuiltinCmd())
	return cmd
}

func newSkillsListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List discovered skills",
		RunE:  runSkillsListCmd,
	}

	cmd.Flags().StringArray("skills-dir", nil, "Skills root directory (repeatable). Defaults: file_state_dir/skills + ~/.claude/skills + ~/.codex/skills")

	return cmd
}

func runSkillsListCmd(cmd *cobra.Command, _ []string) error {
	roots, _ := cmd.Flags().GetStringArray("skills-dir")
	if len(roots) == 0 {
		roots = statepaths.DefaultSkillsRoots()
	}
	list, err := skills.Discover(skills.DiscoverOptions{Roots: roots})
	if err != nil {
		return err
	}

	rows := make([]clifmt.NameDetailRow, 0, len(list))
	for _, skill := range list {
		rows = append(rows, clifmt.NameDetailRow{
			Name:   skill.Name,
			Detail: fmt.Sprintf("id=%s  path=%s", skill.ID, skill.SkillMD),
		})
	}

	clifmt.PrintNameDetailTable(cmd.OutOrStdout(), clifmt.NameDetailTableOptions{
		Title:          "Available skills",
		Rows:           rows,
		EmptyText:      "No skills were discovered.",
		NameHeader:     "NAME",
		DetailHeader:   "DETAILS",
		MinDetailWidth: 48,
	})
	return nil
}
