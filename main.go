package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/yuchou87/claude-hooks/hooks"
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
			os.Exit(hooks.RunCommand(os.Stdin))
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
			return hooks.Install(mode, scope, dryRun)
		},
	}
	cmd.Flags().StringVar(&mode, "mode", "command", "command|http")
	cmd.Flags().StringVar(&scope, "scope", "user", "user|project|local")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print diff without writing")
	return cmd
}
