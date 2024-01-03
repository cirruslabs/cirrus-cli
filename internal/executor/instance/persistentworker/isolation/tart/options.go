package tart

import (
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/logger"
)

type Option func(*Tart)

func WithLogger(logger logger.Lightweight) Option {
	return func(tart *Tart) {
		tart.logger = logger
	}
}

func WithSoftnet() Option {
	return func(tart *Tart) {
		tart.softnet = true
	}
}

func WithDisplay(display string) Option {
	return func(tart *Tart) {
		tart.display = display
	}
}

func WithMountTemporaryWorkingDirectoryFromHost() Option {
	return func(tart *Tart) {
		tart.mountTemporaryWorkingDirectoryFromHost = true
	}
}

func WithVolumes(volumes []*api.Isolation_Tart_Volume) Option {
	return func(tart *Tart) {
		tart.volumes = volumes
	}
}

func WithDiskSize(diskSize uint32) Option {
	return func(tart *Tart) {
		tart.diskSize = diskSize
	}
}
