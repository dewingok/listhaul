package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/dewingok/listhaul/internal/config"
	"github.com/dewingok/listhaul/internal/scheduler"
)

var (
	configPath string
	logLevel   string
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "listhaul",
		Short: "Client-side IMAP inbox cleanup with Sieve-inspired filters",
	}
	root.PersistentFlags().StringVarP(&configPath, "config", "c", config.DefaultConfigPath, "Path to TOML config file")
	root.PersistentFlags().StringVarP(&logLevel, "verbose", "v", "info", "Log level: debug, info, warn, error")

	root.AddCommand(
		newRunCmd(false),
		newRunCmd(true),
		newValidateCmd(),
		newDryRunCmd(),
	)

	return root
}

func newRunCmd(runOnce bool) *cobra.Command {
	use := "run"
	short := "Run the IMAP poll loop"
	if runOnce {
		use = "run-once"
		short = "Run a single poll cycle"
	}

	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, logger, err := loadRuntime()
			if err != nil {
				return err
			}
			return scheduler.Run(cmd.Context(), scheduler.Options{
				Config:  cfg,
				Logger:  logger,
				RunOnce: runOnce,
			})
		},
	}
	return cmd
}

func newValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration without connecting to IMAP",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(configPath)
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stdout, "config valid: %d rule(s), poll every %s, lookback %s\n",
				len(cfg.Rules),
				cfg.Poll.Interval.Duration,
				cfg.Poll.Lookback.Duration,
			)
			return nil
		},
	}
}

func newDryRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "dry-run",
		Short: "Evaluate rules and log actions without mutating mail",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, logger, err := loadRuntime()
			if err != nil {
				return err
			}
			return scheduler.Run(context.Background(), scheduler.Options{
				Config:  cfg,
				Logger:  logger,
				DryRun:  true,
				RunOnce: true,
			})
		},
	}
}

func loadRuntime() (*config.Config, *slog.Logger, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, nil, err
	}

	level := slog.LevelInfo
	switch logLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	case "info":
	default:
		return nil, nil, fmt.Errorf("invalid log level %q", logLevel)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
	return cfg, logger, nil
}
