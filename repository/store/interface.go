package store

import (
	"context"
	"errors"

	"github.com/code19m/sentinel/entity"
)

var ErrNotFound = errors.New("object not found")

type Store interface {
	Add(ctx context.Context, e entity.ErrorInfo) error
	Update(ctx context.Context, e entity.ErrorInfo) error
	FindLast(ctx context.Context, service, operation string, alerted bool) (entity.ErrorInfo, error)
}
