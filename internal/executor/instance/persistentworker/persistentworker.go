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
	"github.com/cirruslabs/cirrus-cli/internal/worker/security"
	"runtime"
	"strings"
)

var ErrInvalidIsolation = errors.New("invalid isolation parameters")

func New(isolation *api.Isolation, security *security.Security, logger logger.Lightweight) (abstract.Instance, error) {
	if isolation == nil {
		nonePolicy := security.NonePolicy()
		if nonePolicy == nil {
			return nil, fmt.Errorf("%w: \"none\" isolation is not allowed by this Persistent Worker's "+
				"security settings", ErrInvalidIsolation)
		}

		return none.New(none.WithLogger(logger))
	}

	switch iso := isolation.Type.(type) {
	case *api.Isolation_None_:
		nonePolicy := security.NonePolicy()
		if nonePolicy == nil {
			return nil, fmt.Errorf("%w: \"none\" isolation is not allowed by this Persistent Worker's "+
				"security settings", ErrInvalidIsolation)
		}

		return none.New(none.WithLogger(logger))
	case *api.Isolation_Parallels_:
		parallelsPolicy := security.ParallelsPolicy()
		if parallelsPolicy == nil {
			return nil, fmt.Errorf("%w: \"parallels\" isolation is not allowed by this Persistent Worker's "+
				"security settings", ErrInvalidIsolation)
		}

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
		containerPolicy := security.ContainerPolicy()
		if containerPolicy == nil {
			return nil, fmt.Errorf("%w: \"container\" isolation is not allowed by this Persistent Worker's "+
				"security settings", ErrInvalidIsolation)
		}

		return container.New(iso.Container.Image, iso.Container.Cpu, iso.Container.Memory, iso.Container.Volumes)
	case *api.Isolation_Tart_:
		tartPolicy := security.TartPolicy()
		if tartPolicy == nil {
			return nil, fmt.Errorf("%w: \"tart\" isolation is not allowed by this Persistent Worker's "+
				"security settings", ErrInvalidIsolation)
		}

		if !tartPolicy.ImageAllowed(iso.Tart.Image) {
			return nil, fmt.Errorf("%w: \"tart\" VM image %q is not allowed by this Persistent Worker's "+
				"security settings", ErrInvalidIsolation, iso.Tart.Image)
		}

		opts := []tart.Option{tart.WithLogger(logger)}

		if iso.Tart.Softnet || tartPolicy.ForceSoftnet {
			opts = append(opts, tart.WithSoftnet())
		}

		if iso.Tart.Display != "" {
			opts = append(opts, tart.WithDisplay(iso.Tart.Display))
		}

		if iso.Tart.MountTemporaryWorkingDirectoryFromHost {
			opts = append(opts, tart.WithMountTemporaryWorkingDirectoryFromHost())
		}

		return tart.New(iso.Tart.Image, iso.Tart.User, iso.Tart.Password, iso.Tart.Cpu, iso.Tart.Memory,
			opts...)
	default:
		return nil, fmt.Errorf("%w: unsupported isolation type %T", ErrInvalidIsolation, iso)
	}
}
