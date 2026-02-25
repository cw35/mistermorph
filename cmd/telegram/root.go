package main

import "github.com/spf13/cobra"

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "telegram",
		Short: "Telegram helper commands",
	}
	cmd.AddCommand(newSendCmd())
	return cmd
}
