package actions

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dewingok/listhaul/internal/config"
)

type MailStore interface {
	FileInto(ctx context.Context, uid uint32, folder string) error
	Discard(ctx context.Context, uid uint32) error
	MarkRead(ctx context.Context, uid uint32) error
	SetFlag(ctx context.Context, uid uint32, flag string, set bool) error
}

type Executor struct {
	store   MailStore
	enabled map[string]struct{}
	logger  *slog.Logger
	dryRun  bool
}

func NewExecutor(store MailStore, enabled map[string]struct{}, logger *slog.Logger, dryRun bool) *Executor {
	if logger == nil {
		logger = slog.Default()
	}
	return &Executor{
		store:   store,
		enabled: enabled,
		logger:  logger,
		dryRun:  dryRun,
	}
}

func (e *Executor) Apply(ctx context.Context, uid uint32, actions []config.ThenAction) error {
	for _, action := range actions {
		if action.Action == "stop" {
			return nil
		}
		if _, ok := e.enabled[action.Action]; !ok {
			return fmt.Errorf("action %q is not enabled", action.Action)
		}
		if err := e.applyOne(ctx, uid, action); err != nil {
			return err
		}
	}
	return nil
}

func (e *Executor) applyOne(ctx context.Context, uid uint32, action config.ThenAction) error {
	if e.dryRun {
		e.logger.Info("dry-run action",
			"uid", uid,
			"action", action.Action,
			"folder", action.Folder,
			"flag", action.Flag,
			"set", action.Set,
		)
		return nil
	}

	switch action.Action {
	case "fileinto":
		return e.store.FileInto(ctx, uid, action.Folder)
	case "discard":
		return e.store.Discard(ctx, uid)
	case "mark_read":
		return e.store.MarkRead(ctx, uid)
	case "flag":
		return e.store.SetFlag(ctx, uid, action.Flag, *action.Set)
	default:
		return fmt.Errorf("unsupported action %q", action.Action)
	}
}
