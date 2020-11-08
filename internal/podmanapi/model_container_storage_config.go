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

// ContainerStorageConfig contains information on the storage configuration of a container.
type ContainerStorageConfig struct {
	// Devices are devices that will be added to the container. Optional.
	Devices []LinuxDevice `json:"devices,omitempty"`
	// Image is the image the container will be based on. The image will be used as the container's root filesystem, and its environment vars, volumes, and other configuration will be applied to the container. Conflicts with Rootfs. At least one of Image or Rootfs must be specified.
	Image string `json:"image,omitempty"`
	// ImageVolumeMode indicates how image volumes will be created. Supported modes are \"ignore\" (do not create), \"tmpfs\" (create as tmpfs), and \"anonymous\" (create as anonymous volumes). The default if unset is anonymous. Optional.
	ImageVolumeMode string `json:"image_volume_mode,omitempty"`
	// Init specifies that an init binary will be mounted into the container, and will be used as PID1.
	Init bool `json:"init,omitempty"`
	// InitPath specifies the path to the init binary that will be added if Init is specified above. If not specified, the default set in the Libpod config will be used. Ignored if Init above is not set. Optional.
	InitPath string `json:"init_path,omitempty"`
	Ipcns *Namespace `json:"ipcns,omitempty"`
	// Mounts are mounts that will be added to the container. These will supersede Image Volumes and VolumesFrom volumes where there are conflicts. Optional.
	Mounts []Mount `json:"mounts,omitempty"`
	// Overlay volumes are named volumes that will be added to the container. Optional.
	OverlayVolumes []OverlayVolume `json:"overlay_volumes,omitempty"`
	// Rootfs is the path to a directory that will be used as the container's root filesystem. No modification will be made to the directory, it will be directly mounted into the container as root. Conflicts with Image. At least one of Image or Rootfs must be specified.
	Rootfs string `json:"rootfs,omitempty"`
	// RootfsPropagation is the rootfs propagation mode for the container. If not set, the default of rslave will be used. Optional.
	RootfsPropagation string `json:"rootfs_propagation,omitempty"`
	// ShmSize is the size of the tmpfs to mount in at /dev/shm, in bytes. Conflicts with ShmSize if IpcNS is not private. Optional.
	ShmSize int64 `json:"shm_size,omitempty"`
	// Volumes are named volumes that will be added to the container. These will supersede Image Volumes and VolumesFrom volumes where there are conflicts. Optional.
	Volumes []NamedVolume `json:"volumes,omitempty"`
	// VolumesFrom is a set of containers whose volumes will be added to this container. The name or ID of the container must be provided, and may optionally be followed by a : and then one or more comma-separated options. Valid options are 'ro', 'rw', and 'z'. Options will be used for all volumes sourced from the container.
	VolumesFrom []string `json:"volumes_from,omitempty"`
	// WorkDir is the container's working directory. If unset, the default, /, will be used. Optional.
	WorkDir string `json:"work_dir,omitempty"`
}
