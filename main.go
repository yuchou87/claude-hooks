package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "claude-hooks",
		Short: "Claude Code hook framework",
	}

	root.AddCommand(newRunCmd())
	root.AddCommand(newInstallCmd())

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "Process a hook event from stdin (command mode)",
		Run: func(cmd *cobra.Command, args []string) {
			// hooks package filled in Task 7
			os.Exit(0)
		},
	}
}

func newInstallCmd() *cobra.Command {
	var mode, scope string
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Register claude-hooks in ~/.claude/settings.json",
		RunE: func(cmd *cobra.Command, args []string) error {
			// hooks package filled in Task 9
			return nil
		},
	}
	cmd.Flags().StringVar(&mode, "mode", "command", "command|http")
	cmd.Flags().StringVar(&scope, "scope", "user", "user|project|local")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print diff without writing")
	return cmd
}
