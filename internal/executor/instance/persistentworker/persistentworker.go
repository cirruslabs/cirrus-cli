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
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/persistentworker/isolation/vetu"
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
		return newTart(iso, security, logger)
	case *api.Isolation_Vetu_:
		return newVetu(iso, security, logger)
	default:
		return nil, fmt.Errorf("%w: unsupported isolation type %T", ErrInvalidIsolation, iso)
	}
}

func newTart(iso *api.Isolation_Tart_, security *security.Security, logger logger.Lightweight) (*tart.Tart, error) {
	tartPolicy := security.TartPolicy()
	if tartPolicy == nil {
		return nil, fmt.Errorf("%w: \"tart\" isolation is not allowed by this Persistent Worker's "+
			"security settings", ErrInvalidIsolation)
	}

	if !tartPolicy.AllowedImages.ImageAllowed(iso.Tart.Image) {
		return nil, fmt.Errorf("%w: \"tart\" VM image %q is not allowed by this Persistent Worker's "+
			"security settings", ErrInvalidIsolation, iso.Tart.Image)
	}

	for _, volume := range iso.Tart.Volumes {
		if !tartPolicy.VolumeAllowed(volume) {
			var volumeIdent string

			if volume.Name != "" {
				volumeIdent = volume.Name
			} else {
				volumeIdent = volume.Source
			}

			return nil, fmt.Errorf("%w: volume %q is not allowed by this Persistent Worker's "+
				"security settings", ErrInvalidIsolation, volumeIdent)
		}
	}

	opts := []tart.Option{tart.WithLogger(logger)}

	if iso.Tart.Softnet || tartPolicy.ForceSoftnet {
		opts = append(opts, tart.WithSoftnet())
	}

	return tart.NewFromIsolation(iso, opts...)
}

func newVetu(iso *api.Isolation_Vetu_, security *security.Security, logger logger.Lightweight) (*vetu.Vetu, error) {
	vetuPolicy := security.VetuPolicy()
	if vetuPolicy == nil {
		return nil, fmt.Errorf("%w: \"vetu\" isolation is not allowed by this Persistent Worker's "+
			"security settings", ErrInvalidIsolation)
	}

	if !vetuPolicy.AllowedImages.ImageAllowed(iso.Vetu.Image) {
		return nil, fmt.Errorf("%w: \"vetu\" VM image %q is not allowed by this Persistent Worker's "+
			"security settings", ErrInvalidIsolation, iso.Vetu.Image)
	}

	opts := []vetu.Option{vetu.WithLogger(logger)}

	switch networking := iso.Vetu.Networking.(type) {
	case *api.Isolation_Vetu_Bridged_:
		opts = append(opts, vetu.WithBridgedInterface(networking.Bridged.Interface))
	case *api.Isolation_Vetu_Host_:
		opts = append(opts, vetu.WithHostNetworking())
	default:
		// use default gVisor-backed networking
	}

	return vetu.NewFromIsolation(iso, opts...)
}
