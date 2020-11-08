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

// HostInfo describes the libpod host
type HostInfo struct {
	Arch string `json:"arch,omitempty"`
	BuildahVersion string `json:"buildahVersion,omitempty"`
	CgroupManager string `json:"cgroupManager,omitempty"`
	CgroupVersion string `json:"cgroupVersion,omitempty"`
	Conmon *ConmonInfo `json:"conmon,omitempty"`
	Cpus int64 `json:"cpus,omitempty"`
	Distribution *DistributionInfo `json:"distribution,omitempty"`
	EventLogger string `json:"eventLogger,omitempty"`
	Hostname string `json:"hostname,omitempty"`
	IdMappings *IdMappings `json:"idMappings,omitempty"`
	Kernel string `json:"kernel,omitempty"`
	Linkmode string `json:"linkmode,omitempty"`
	MemFree int64 `json:"memFree,omitempty"`
	MemTotal int64 `json:"memTotal,omitempty"`
	OciRuntime *OciRuntimeInfo `json:"ociRuntime,omitempty"`
	Os string `json:"os,omitempty"`
	RemoteSocket *RemoteSocket `json:"remoteSocket,omitempty"`
	Rootless bool `json:"rootless,omitempty"`
	RuntimeInfo map[string]interface{} `json:"runtimeInfo,omitempty"`
	Slirp4netns *SlirpInfo `json:"slirp4netns,omitempty"`
	SwapFree int64 `json:"swapFree,omitempty"`
	SwapTotal int64 `json:"swapTotal,omitempty"`
	Uptime string `json:"uptime,omitempty"`
}
