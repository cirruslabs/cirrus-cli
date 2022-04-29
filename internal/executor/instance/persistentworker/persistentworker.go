package persistentworker

import (
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/abstract"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/persistentworker/isolation/container"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/persistentworker/isolation/none"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/persistentworker/isolation/parallels"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/persistentworker/isolation/tart"
	"github.com/cirruslabs/cirrus-cli/internal/logger"
	"runtime"
	"strings"
)

var ErrInvalidIsolation = errors.New("invalid isolation parameters")

func New(isolation *api.Isolation, logger logger.Lightweight) (abstract.Instance, error) {
	if isolation == nil {
		return none.New(none.WithLogger(logger))
	}

	switch iso := isolation.Type.(type) {
	case *api.Isolation_None_:
		return none.New(none.WithLogger(logger))
	case *api.Isolation_Parallels_:
		if iso.Parallels.Platform != api.Platform_DARWIN && iso.Parallels.Platform != api.Platform_LINUX {
			return nil, fmt.Errorf("%w: only Darwin and Linux are currently supported for Parallels",
				ErrInvalidIsolation)
		}
		if runtime.GOARCH != "amd64" {
			return nil, fmt.Errorf("%w: only Intel (amd64) is currently supported for Parallels, "+
				"use Tart if you're running on Apple silicon (arm64)", ErrInvalidIsolation)
		}

		return parallels.New(iso.Parallels.Image, iso.Parallels.User, iso.Parallels.Password,
			strings.ToLower(iso.Parallels.Platform.String()), parallels.WithLogger(logger))
	case *api.Isolation_Container_:
		return container.New(iso.Container.Image, iso.Container.Cpu, iso.Container.Memory, iso.Container.Volumes)
	case *api.Isolation_Tart_:
		return tart.New(iso.Tart.Image, iso.Tart.User, iso.Tart.Password, iso.Tart.Cpu, iso.Tart.Memory,
			tart.WithLogger(logger))
	default:
		return nil, fmt.Errorf("%w: unsupported isolation type %T", ErrInvalidIsolation, iso)
	}
}
