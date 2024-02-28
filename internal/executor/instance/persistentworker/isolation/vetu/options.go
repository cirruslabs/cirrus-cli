package vetu

import (
	"github.com/cirruslabs/cirrus-cli/internal/logger"
)

type Option func(*Vetu)

func WithLogger(logger logger.Lightweight) Option {
	return func(tart *Vetu) {
		tart.logger = logger
	}
}

func WithBridgedInterface(bridgedInterface string) Option {
	return func(tart *Vetu) {
		tart.BridgedInterface = bridgedInterface
	}
}

func WithHostNetworking() Option {
	return func(tart *Vetu) {
		tart.HostNetworking = true
	}
}

func WithDiskSize(diskSize uint32) Option {
	return func(vetu *Vetu) {
		vetu.DiskSize = diskSize
	}
}
