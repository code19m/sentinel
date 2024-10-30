package usecase

import (
	"context"

	"github.com/code19m/sentinel/entity"
)

type UseCase interface {
	SendError(ctx context.Context, e entity.ErrorInfo) error
}
