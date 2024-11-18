package abstract

import (
	"context"
	"errors"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/runconfig"
	"github.com/cirruslabs/echelon"
	"go.opentelemetry.io/otel/attribute"
	"time"
)

type Instance interface {
	Run(ctx context.Context, config *runconfig.RunConfig) error
	WorkingDirectory(projectDir string, dirtyMode bool) string
	Close(ctx context.Context) error
	Attributes() []attribute.KeyValue
}

var (
	ErrWarmupScriptFailed = errors.New("warm-up script failed")
	ErrWarmupTimeout      = errors.New("warm-up script timed out")
)

type WarmableInstance interface {
	// Warmup can be optionally called in case of a persistent worker is configured to be warm
	Warmup(
		ctx context.Context,
		ident string,
		env map[string]string,
		warmupScript string,
		warmupTimeout time.Duration,
		logger *echelon.Logger,
	) error
}
