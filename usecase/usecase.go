package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/code19m/sentinel/entity"
	"github.com/code19m/sentinel/repository/notifier"
	"github.com/code19m/sentinel/repository/store"
)

func New(log *slog.Logger, store store.Store, notifier notifier.Notifier, alertCooldownMinutes int) usecase {
	return usecase{
		log:                  log,
		store:                store,
		notifier:             notifier,
		alertCooldownMinutes: alertCooldownMinutes,
	}
}

type usecase struct {
	log      *slog.Logger
	store    store.Store
	notifier notifier.Notifier

	alertCooldownMinutes int
}

func (uc usecase) SendError(ctx context.Context, e entity.ErrorInfo) error {
	err := uc.store.Add(ctx, e)
	if err != nil {
		return fmt.Errorf("usecase.SendError: %w", err)
	}

	go uc.handleAlert(ctx, e)

	return nil
}

func (uc usecase) handleAlert(ctx context.Context, e entity.ErrorInfo) {
	ctx = context.Background()

	lastAlerted, err := uc.store.FindLast(ctx, e.Service, e.Operation, true)
	if err != nil && err != store.ErrNotFound {
		uc.log.ErrorContext(ctx, fmt.Sprintf("usecase.handleAlert: %v", err))
		return
	}

	// Skip alerting if the last alert was sent less than AlertCooldownMinutes ago
	if err == nil && time.Since(lastAlerted.CreatedAt) < time.Minute*time.Duration(uc.alertCooldownMinutes) {
		return
	}

	err = uc.notifier.Notify(ctx, e)
	if err != nil {
		uc.log.ErrorContext(ctx, fmt.Sprintf("usecase.handleAlert: %v", err))
		return
	}

	e.Alerted = true
	err = uc.store.Update(ctx, e)
	if err != nil {
		uc.log.ErrorContext(ctx, fmt.Sprintf("usecase.handleAlert: %v", err))
		return
	}
}
