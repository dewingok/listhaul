package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dewingok/listhaul/internal/actions"
	"github.com/dewingok/listhaul/internal/config"
	"github.com/dewingok/listhaul/internal/filter"
	"github.com/dewingok/listhaul/internal/imapclient"
	"github.com/dewingok/listhaul/internal/state"
)

type Options struct {
	Config  *config.Config
	Logger  *slog.Logger
	DryRun  bool
	RunOnce bool
}

func Run(ctx context.Context, opts Options) error {
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}

	var store *state.Store
	if opts.Config.State.Enabled && !opts.DryRun {
		var err error
		store, err = state.New(opts.Config.State.Path)
		if err != nil {
			return err
		}
		if err := store.Load(); err != nil {
			return err
		}
		defer func() {
			if err := store.Save(); err != nil {
				opts.Logger.Error("save state", "error", err)
			}
		}()
	}

	runCycle := func() error {
		return runPollCycle(ctx, opts, store)
	}

	if opts.RunOnce {
		return runCycle()
	}

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	ticker := time.NewTicker(opts.Config.Poll.Interval.Duration)
	defer ticker.Stop()

	opts.Logger.Info("starting poll loop",
		"interval", opts.Config.Poll.Interval.Duration,
		"lookback", opts.Config.Poll.Lookback.Duration,
		"mailbox", opts.Config.Account.Mailbox,
		"dry_run", opts.DryRun,
	)

	if err := runCycle(); err != nil {
		opts.Logger.Error("poll cycle failed", "error", err)
	}

	for {
		select {
		case <-ctx.Done():
			opts.Logger.Info("shutting down")
			return nil
		case <-ticker.C:
			if err := runCycle(); err != nil {
				opts.Logger.Error("poll cycle failed", "error", err)
			}
		}
	}
}

func runPollCycle(ctx context.Context, opts Options, store *state.Store) error {
	client, err := imapclient.Connect(ctx, opts.Config)
	if err != nil {
		return err
	}
	defer client.Close()

	uids, err := client.SearchRecent(ctx)
	if err != nil {
		return err
	}
	opts.Logger.Info("found candidate messages", "count", len(uids))

	if len(uids) == 0 {
		return nil
	}

	mailbox := opts.Config.Account.Mailbox
	filteredUIDs := make([]uint32, 0, len(uids))
	for _, uid := range uids {
		if store != nil && store.Seen(mailbox, uid) {
			continue
		}
		filteredUIDs = append(filteredUIDs, uid)
	}

	if len(filteredUIDs) == 0 {
		opts.Logger.Info("all candidate messages already processed")
		return nil
	}

	headersByUID, err := client.FetchHeaders(ctx, filteredUIDs)
	if err != nil {
		return err
	}

	executor := actions.NewExecutor(client, opts.Config.EnabledActions(), opts.Logger, opts.DryRun)

	for _, uid := range filteredUIDs {
		headers, ok := headersByUID[uid]
		if !ok {
			opts.Logger.Warn("missing headers", "uid", uid)
			continue
		}

		match, matched := filter.EvaluateRules(opts.Config.Rules, headers)
		if !matched {
			continue
		}

		opts.Logger.Info("rule matched",
			"uid", uid,
			"rule", ruleName(match.RuleName),
			"actions", len(match.Actions),
			"dry_run", opts.DryRun,
		)

		if err := executor.Apply(ctx, uid, match.Actions); err != nil {
			return fmt.Errorf("apply actions for uid %d: %w", uid, err)
		}

		if store != nil {
			store.Mark(mailbox, uid)
		}
	}

	return nil
}

func ruleName(name string) string {
	if name == "" {
		return "(unnamed)"
	}
	return name
}
