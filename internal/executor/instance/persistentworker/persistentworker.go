package persistentworker

import (
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/abstract"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/persistentworker/isolation/none"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/persistentworker/isolation/parallels"
	"github.com/cirruslabs/cirrus-cli/internal/logger"
	"strings"
)

var ErrInvalidIsolation = errors.New("invalid isolation parameters")

func New(isolation *api.Isolation, logger logger.Lightweight) (abstract.Instance, error) {
	if isolation == nil {
		return none.New()
	}

	switch iso := isolation.Type.(type) {
	case *api.Isolation_None_:
		return none.New()
	case *api.Isolation_Parallels_:
		if iso.Parallels.Platform != api.Platform_DARWIN && iso.Parallels.Platform != api.Platform_LINUX {
			return nil, fmt.Errorf("%w: only Darwin and Linux are currently supported", ErrInvalidIsolation)
		}

		return parallels.New(iso.Parallels.Image, iso.Parallels.User, iso.Parallels.Password,
			strings.ToLower(iso.Parallels.Platform.String()), parallels.WithLogger(logger))
	default:
		return nil, fmt.Errorf("%w: unsupported isolation type %T", ErrInvalidIsolation, iso)
	}
}
