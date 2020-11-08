/*
 * Provides a container compatible interface.
 *
 * This documentation describes the Podman v2.0 RESTful API. It replaces the Podman v1.0 API and was initially delivered along with Podman v2.0.  It consists of a Docker-compatible API and a Libpod API providing support for Podman’s unique features such as pods.  To start the service and keep it running for 5,000 seconds (-t 0 runs forever):  podman system service -t 5000 &  You can then use cURL on the socket using requests documented below.  NOTE: if you install the package podman-docker, it will create a symbolic link for /var/run/docker.sock to /run/podman/podman.sock  See podman-service(1) for more information.  Quick Examples:  'podman info'  curl --unix-socket /run/podman/podman.sock http://d/v1.0.0/libpod/info  'podman pull quay.io/containers/podman'  curl -XPOST --unix-socket /run/podman/podman.sock -v 'http://d/v1.0.0/images/create?fromImage=quay.io%2Fcontainers%2Fpodman'  'podman list images'  curl --unix-socket /run/podman/podman.sock -v 'http://d/v1.0.0/libpod/images/json' | jq
 *
 * API version: 0.0.1
 * Contact: podman@lists.podman.io
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */

package swagger

type ContainerResourceConfig struct {
	// OOMScoreAdj adjusts the score used by the OOM killer to determine processes to kill for the container's process. Optional.
	OomScoreAdj int64 `json:"oom_score_adj,omitempty"`
	// Rlimits are POSIX rlimits to apply to the container. Optional.
	RLimits []PosixRlimit `json:"r_limits,omitempty"`
	ResourceLimits *LinuxResources `json:"resource_limits,omitempty"`
	// IO read rate limit per cgroup per device, bytes per second
	ThrottleReadBpsDevice map[string]LinuxThrottleDevice `json:"throttleReadBpsDevice,omitempty"`
	// IO read rate limit per cgroup per device, IO per second
	ThrottleReadIOPSDevice map[string]LinuxThrottleDevice `json:"throttleReadIOPSDevice,omitempty"`
	// IO write rate limit per cgroup per device, bytes per second
	ThrottleWriteBpsDevice map[string]LinuxThrottleDevice `json:"throttleWriteBpsDevice,omitempty"`
	// IO write rate limit per cgroup per device, IO per second
	ThrottleWriteIOPSDevice map[string]LinuxThrottleDevice `json:"throttleWriteIOPSDevice,omitempty"`
	// CgroupConf are key-value options passed into the container runtime that are used to configure cgroup v2. Optional.
	Unified map[string]string `json:"unified,omitempty"`
	// Weight per cgroup per device, can override BlkioWeight
	WeightDevice map[string]LinuxWeightDevice `json:"weightDevice,omitempty"`
}
