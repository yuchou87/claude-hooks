package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/yuchou87/claude-hooks/hooks"
	_ "github.com/yuchou87/claude-hooks/rules" // register all rules via init()
)

func main() {
	root := &cobra.Command{
		Use:   "claude-hooks",
		Short: "Claude Code hook framework",
	}

	root.AddCommand(newRunCmd())
	root.AddCommand(newInstallCmd())
	root.AddCommand(newServeCmd())

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
	var mode, scope, addr string
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Register claude-hooks in ~/.claude/settings.json",
		RunE: func(cmd *cobra.Command, args []string) error {
			return hooks.Install(mode, scope, addr, dryRun)
		},
	}
	cmd.Flags().StringVar(&mode, "mode", "command", "command|http")
	cmd.Flags().StringVar(&scope, "scope", "user", "user|project|local")
	cmd.Flags().StringVar(&addr, "addr", "127.0.0.1:8787", "daemon listen address (http mode only; must match serve --addr)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print diff without writing")
	return cmd
}

func newServeCmd() *cobra.Command {
	var addr, configPath, scriptsDir string
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run HTTP daemon for remote approval (http mode)",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Resolve defaults for config + scripts from user home dir
			if configPath == "" || scriptsDir == "" {
				home, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("cannot determine home directory: %w", err)
				}
				if configPath == "" {
					configPath = filepath.Join(home, ".claude-hooks", "config.yaml")
				}
				if scriptsDir == "" {
					scriptsDir = filepath.Join(home, ".claude-hooks", "scripts")
				}
			}

			// Initial load of dynamic rules (YAML + scripts)
			rules, err := hooks.BuildDynamicRules(configPath, scriptsDir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "claude-hooks: warning: failed to load rules: %v\n", err)
				// Non-fatal: serve with only native Go rules
			} else {
				hooks.StoreDynamic(rules)
				fmt.Fprintf(os.Stderr, "claude-hooks: loaded %d dynamic rule(s)\n", len(rules))
			}

			// Start file watcher for hot-reload
			stopWatcher, watchErr := hooks.StartWatcher(configPath, scriptsDir, func() {
				newRules, err := hooks.BuildDynamicRules(configPath, scriptsDir)
				if err != nil {
					fmt.Fprintf(os.Stderr, "claude-hooks: hot-reload failed (keeping old rules): %v\n", err)
					return // keep old rules on error
				}
				hooks.StoreDynamic(newRules)
				fmt.Fprintf(os.Stderr, "claude-hooks: hot-reloaded %d dynamic rule(s)\n", len(newRules))
			})
			if watchErr != nil {
				fmt.Fprintf(os.Stderr, "claude-hooks: warning: watcher failed to start: %v\n", watchErr)
				stopWatcher = func() {} // no-op
			}

			approver := hooks.NewApprover()
			srv := hooks.NewServer(addr, approver)

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

			errCh := make(chan error, 1)
			go func() { errCh <- srv.ListenAndServe() }()

			fmt.Fprintf(os.Stderr, "claude-hooks: serving on %s\n", addr)

			select {
			case sig := <-sigCh:
				fmt.Fprintf(os.Stderr, "claude-hooks: received %s, shutting down\n", sig)
				stopWatcher()
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				return srv.Shutdown(ctx)
			case err := <-errCh:
				stopWatcher()
				return err
			}
		},
	}
	cmd.Flags().StringVar(&addr, "addr", "127.0.0.1:8787", "listen address (loopback only)")
	cmd.Flags().StringVar(&configPath, "config", "", "YAML rules file (default: ~/.claude-hooks/config.yaml)")
	cmd.Flags().StringVar(&scriptsDir, "scripts-dir", "", "scripts directory (default: ~/.claude-hooks/scripts)")
	return cmd
}
