package resolver

import "github.com/cirruslabs/cirrus-cli/internal/agent/executor/metrics/source/cgroup/subsystem"

type Resolver interface {
	Resolve(subsystemName subsystem.SubsystemName) (string, string, error)
}
