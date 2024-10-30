package notifier

import (
	"context"

	"github.com/code19m/sentinel/entity"
)

type Notifier interface {
	Notify(ctx context.Context, e entity.ErrorInfo) error
}
