package store

import (
	"context"
	"errors"

	"github.com/code19m/sentinel/entity"
)

var ErrNotFound = errors.New("object not found")
var ErrAlertCooldown = errors.New("alert is in cooldown period")

type Store interface {
	Add(ctx context.Context, e entity.ErrorInfo) error
	Update(ctx context.Context, e entity.ErrorInfo) error
	CheckAndMarkAlerted(ctx context.Context, e entity.ErrorInfo, cooldownMinutes int) error
	GetErrorFrequency(ctx context.Context, service, operation string, minutesBack int) (int, error)
}
