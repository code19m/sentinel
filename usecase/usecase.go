package usecase

import (
	"context"
	"fmt"
	"log/slog"

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

	go func() {
		alertCtx := context.Background()

		err := uc.store.CheckAndMarkAlerted(alertCtx, e, uc.alertCooldownMinutes)
		if err != nil {
			if err == store.ErrAlertCooldown {
				return
			}
			uc.log.ErrorContext(alertCtx, fmt.Sprintf("usecase.SendError: CheckAndMarkAlerted failed: %v", err))
			return
		}

		frequency, err := uc.store.GetErrorFrequency(alertCtx, e.Service, e.Operation, uc.alertCooldownMinutes)
		if err != nil {
			uc.log.ErrorContext(alertCtx, fmt.Sprintf("usecase.SendError: GetErrorFrequency failed: %v", err))
		} else {
			e.Frequency = frequency
			e.FrequencyMinutes = uc.alertCooldownMinutes
		}

		err = uc.notifier.Notify(alertCtx, e)
		if err != nil {
			uc.log.ErrorContext(alertCtx, fmt.Sprintf("usecase.SendError: notification failed: %v", err))
			return
		}
	}()

	return nil
}
