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

// NetworkConfig configures the network namespace for the container
type NetworkConfig struct {
	DNSOpt []string `json:"DNSOpt,omitempty"`
	DNSSearch []string `json:"DNSSearch,omitempty"`
	DNSServers []string `json:"DNSServers,omitempty"`
	ExposedPorts map[string]interface{} `json:"ExposedPorts,omitempty"`
	HTTPProxy bool `json:"HTTPProxy,omitempty"`
	IP6Address string `json:"IP6Address,omitempty"`
	IPAddress string `json:"IPAddress,omitempty"`
	LinkLocalIP []string `json:"LinkLocalIP,omitempty"`
	MacAddress string `json:"MacAddress,omitempty"`
	NetMode *NetworkMode `json:"NetMode,omitempty"`
	Network string `json:"Network,omitempty"`
	NetworkAlias []string `json:"NetworkAlias,omitempty"`
	PortBindings *PortMap `json:"PortBindings,omitempty"`
	Publish []string `json:"Publish,omitempty"`
	PublishAll bool `json:"PublishAll,omitempty"`
}
