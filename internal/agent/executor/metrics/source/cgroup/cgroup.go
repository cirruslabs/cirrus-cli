package cgroup

import "errors"

var (
	ErrUnconfigured        = errors.New("cgroup is not configured for this process")
	ErrUnsupportedPlatform = errors.New("cgroup is not supported on this platform")
)
