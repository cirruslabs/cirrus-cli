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
import (
	"time"
)

type InlineResponse20015 struct {
	// Anonymous indicates that the volume was created as an anonymous volume for a specific container, and will be be removed when any container using it is removed.
	Anonymous bool `json:"Anonymous,omitempty"`
	// CreatedAt is the date and time the volume was created at. This is not stored for older Libpod volumes; if so, it will be omitted.
	CreatedAt time.Time `json:"CreatedAt,omitempty"`
	// Driver is the driver used to create the volume. This will be properly implemented in a future version.
	Driver string `json:"Driver,omitempty"`
	// GID is the GID that the volume was created with.
	GID int64 `json:"GID,omitempty"`
	// Labels includes the volume's configured labels, key:value pairs that can be passed during volume creation to provide information for third party tools.
	Labels map[string]string `json:"Labels,omitempty"`
	// Mountpoint is the path on the host where the volume is mounted.
	Mountpoint string `json:"Mountpoint,omitempty"`
	// Name is the name of the volume.
	Name string `json:"Name,omitempty"`
	// Options is a set of options that were used when creating the volume. It is presently not used.
	Options map[string]string `json:"Options,omitempty"`
	// Scope is unused and provided solely for Docker compatibility. It is unconditionally set to \"local\".
	Scope string `json:"Scope,omitempty"`
	// Status is presently unused and provided only for Docker compatibility. In the future it will be used to return information on the volume's current state.
	Status map[string]string `json:"Status,omitempty"`
	// UID is the UID that the volume was created with.
	UID int64 `json:"UID,omitempty"`
}
