package source

import (
	"context"
	"time"
)

type CPU interface {
	Name() string
	NumCpusUsed(ctx context.Context, pollInterval time.Duration) (float64, error)
}

type Memory interface {
	Name() string
	AmountMemoryUsed(ctx context.Context) (float64, error)
}
