package resolver

import (
	"errors"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/agent/executor/metrics/source/cgroup/subsystem"
	"github.com/prometheus/procfs"
	"path/filepath"
)

type LinuxResolver struct {
	mountInfos []*procfs.MountInfo
	cgroups    []procfs.Cgroup
}

type Mount struct {
	Root       string
	MountPoint string
}

const (
	v1FSType            = "cgroup"
	v2FSType            = "cgroup2"
	preferredMountpoint = "/sys/fs/cgroup"
)

var (
	ErrInitFailed = errors.New("failed to initialize cgroup resolver")
	ErrInternal   = errors.New("internal cgroup resolver error")
)

func New() (*LinuxResolver, error) {
	self, err := procfs.Self()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to open /proc/self: %v", ErrInitFailed, err)
	}

	mountInfos, err := self.MountInfo()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to retrieve mounts: %v", ErrInitFailed, err)
	}

	cgroups, err := self.Cgroups()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to retrieve cgroups: %v", ErrInitFailed, err)
	}

	return &LinuxResolver{
		mountInfos: mountInfos,
		cgroups:    cgroups,
	}, nil
}

func (resolver *LinuxResolver) Resolve(subsystemName subsystem.SubsystemName) (string, string, error) {
	// Determine where a cgroup hierarchy for a given subsystem is mounted (e.g. /sys/fs/cgroup/cpuset)
	var v1mount *Mount
	var v2mount *Mount

	for _, mount := range resolver.mountInfos {
		switch mount.FSType {
		case v1FSType:
			if _, ok := mount.SuperOptions[string(subsystemName)]; !ok {
				continue
			}

			if v1mount != nil && v1mount.MountPoint == preferredMountpoint {
				continue
			}

			v1mount = &Mount{Root: mount.Root, MountPoint: mount.MountPoint}
		case v2FSType:
			if v2mount != nil && v2mount.MountPoint == preferredMountpoint {
				continue
			}

			v2mount = &Mount{Root: mount.Root, MountPoint: mount.MountPoint}
		}
	}

	// Determine the path within a cgroup hierarchy where our process is placed
	// (e.g. /docker/e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855)
	var pathWithinV1Hierarchy string
	var pathWithinV2Hierarchy string

	for _, cgroup := range resolver.cgroups {
		// From /proc/[pid]/cgroup's documentation[1] on hierarchy ID field:
		//
		// >For the cgroups version 2 hierarchy, this field contains the value 0.
		//
		// From /proc/[pid]/cgroup's documentation[1] on controllers field:
		//
		// >For the cgroups version 2 hierarchy, this field is empty.
		//
		// [1]: https://man7.org/linux/man-pages/man7/cgroups.7.html
		if cgroup.HierarchyID == 0 && len(cgroup.Controllers) == 0 {
			pathWithinV2Hierarchy = cgroup.Path
		}

		for _, controller := range cgroup.Controllers {
			if controller == string(subsystemName) {
				pathWithinV1Hierarchy = cgroup.Path
			}
		}
	}

	var subsystemPathV1 string
	var subsystemPathV2 string

	// Determine the final path for a given subsystem, preferring cgroup version 1
	if pathWithinV1Hierarchy != "" {
		if v1mount == nil {
			return "", "", fmt.Errorf("%w: process uses cgroup version 1, yet it's hierarchy is not mounted",
				ErrInternal)
		}

		normalizedPath, err := filepath.Rel(v1mount.Root, pathWithinV1Hierarchy)
		if err != nil {
			return "", "", fmt.Errorf("%w: failed to normalize subsystem path: %v", ErrInternal, err)
		}

		subsystemPathV1 = filepath.Join(v1mount.MountPoint, normalizedPath)
	} else if pathWithinV2Hierarchy != "" {
		if v2mount == nil {
			return "", "", fmt.Errorf("%w: process uses cgroup version 2, yet it's hierarchy is not mounted",
				ErrInternal)
		}

		normalizedPath, err := filepath.Rel(v2mount.Root, pathWithinV2Hierarchy)
		if err != nil {
			return "", "", fmt.Errorf("%w: failed to normalize subsystem path: %v", ErrInternal, err)
		}

		subsystemPathV2 = filepath.Join(v2mount.MountPoint, normalizedPath)
	}

	return subsystemPathV1, subsystemPathV2, nil
}
