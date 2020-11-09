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

type Body4 struct {
	// Attach to stderr of the exec command
	AttachStderr bool `json:"AttachStderr,omitempty"`
	// Attach to stdin of the exec command
	AttachStdin bool `json:"AttachStdin,omitempty"`
	// Attach to stdout of the exec command
	AttachStdout bool `json:"AttachStdout,omitempty"`
	// Command to run, as a string or array of strings.
	Cmd []string `json:"Cmd,omitempty"`
	// \"Override the key sequence for detaching a container. Format is a single character [a-Z] or ctrl-<value> where <value> is one of: a-z, @, ^, [, , or _.\" 
	DetachKeys string `json:"DetachKeys,omitempty"`
	// A list of environment variables in the form [\"VAR=value\", ...]
	Env []string `json:"Env,omitempty"`
	// Runs the exec process with extended privileges
	Privileged bool `json:"Privileged,omitempty"`
	// Allocate a pseudo-TTY
	Tty bool `json:"Tty,omitempty"`
	// \"The user, and optionally, group to run the exec process inside the container. Format is one of: user, user:group, uid, or uid:gid.\" 
	User string `json:"User,omitempty"`
	// The working directory for the exec process inside the container.
	WorkingDir string `json:"WorkingDir,omitempty"`
}
