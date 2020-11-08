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

// Here, \"non-portable\" means \"dependent of the host we are running on\". Portable information *should* appear in Config.
type HostConfig struct {
	AutoRemove bool `json:"AutoRemove,omitempty"`
	// Applicable to all platforms
	Binds []string `json:"Binds,omitempty"`
	BlkioDeviceReadBps []ThrottleDevice `json:"BlkioDeviceReadBps,omitempty"`
	BlkioDeviceReadIOps []ThrottleDevice `json:"BlkioDeviceReadIOps,omitempty"`
	BlkioDeviceWriteBps []ThrottleDevice `json:"BlkioDeviceWriteBps,omitempty"`
	BlkioDeviceWriteIOps []ThrottleDevice `json:"BlkioDeviceWriteIOps,omitempty"`
	BlkioWeight int32 `json:"BlkioWeight,omitempty"`
	BlkioWeightDevice []WeightDevice `json:"BlkioWeightDevice,omitempty"`
	CapAdd *StrSlice `json:"CapAdd,omitempty"`
	CapDrop *StrSlice `json:"CapDrop,omitempty"`
	Cgroup *CgroupSpec `json:"Cgroup,omitempty"`
	// Applicable to UNIX platforms
	CgroupParent string `json:"CgroupParent,omitempty"`
	CgroupnsMode *CgroupnsMode `json:"CgroupnsMode,omitempty"`
	// Applicable to Windows
	ConsoleSize []int32 `json:"ConsoleSize,omitempty"`
	ContainerIDFile string `json:"ContainerIDFile,omitempty"`
	// Applicable to Windows
	CpuCount int64 `json:"CpuCount,omitempty"`
	CpuPercent int64 `json:"CpuPercent,omitempty"`
	CpuPeriod int64 `json:"CpuPeriod,omitempty"`
	CpuQuota int64 `json:"CpuQuota,omitempty"`
	CpuRealtimePeriod int64 `json:"CpuRealtimePeriod,omitempty"`
	CpuRealtimeRuntime int64 `json:"CpuRealtimeRuntime,omitempty"`
	// Applicable to all platforms
	CpuShares int64 `json:"CpuShares,omitempty"`
	CpusetCpus string `json:"CpusetCpus,omitempty"`
	CpusetMems string `json:"CpusetMems,omitempty"`
	DeviceCgroupRules []string `json:"DeviceCgroupRules,omitempty"`
	DeviceRequests []DeviceRequest `json:"DeviceRequests,omitempty"`
	Devices []DeviceMapping `json:"Devices,omitempty"`
	Dns []string `json:"Dns,omitempty"`
	DnsOptions []string `json:"DnsOptions,omitempty"`
	DnsSearch []string `json:"DnsSearch,omitempty"`
	ExtraHosts []string `json:"ExtraHosts,omitempty"`
	GroupAdd []string `json:"GroupAdd,omitempty"`
	IOMaximumBandwidth int32 `json:"IOMaximumBandwidth,omitempty"`
	IOMaximumIOps int32 `json:"IOMaximumIOps,omitempty"`
	// Run a custom init inside the container, if null, use the daemon's configured settings
	Init bool `json:"Init,omitempty"`
	IpcMode *IpcMode `json:"IpcMode,omitempty"`
	Isolation *Isolation `json:"Isolation,omitempty"`
	KernelMemory int64 `json:"KernelMemory,omitempty"`
	KernelMemoryTCP int64 `json:"KernelMemoryTCP,omitempty"`
	Links []string `json:"Links,omitempty"`
	LogConfig *LogConfig `json:"LogConfig,omitempty"`
	// MaskedPaths is the list of paths to be masked inside the container (this overrides the default set of paths)
	MaskedPaths []string `json:"MaskedPaths,omitempty"`
	Memory int64 `json:"Memory,omitempty"`
	MemoryReservation int64 `json:"MemoryReservation,omitempty"`
	MemorySwap int64 `json:"MemorySwap,omitempty"`
	MemorySwappiness int64 `json:"MemorySwappiness,omitempty"`
	// Mounts specs used by the container
	Mounts []Mount `json:"Mounts,omitempty"`
	NanoCpus int64 `json:"NanoCpus,omitempty"`
	NetworkMode *NetworkMode `json:"NetworkMode,omitempty"`
	OomKillDisable bool `json:"OomKillDisable,omitempty"`
	OomScoreAdj int64 `json:"OomScoreAdj,omitempty"`
	PidMode *PidMode `json:"PidMode,omitempty"`
	PidsLimit int64 `json:"PidsLimit,omitempty"`
	PortBindings *PortMap `json:"PortBindings,omitempty"`
	Privileged bool `json:"Privileged,omitempty"`
	PublishAllPorts bool `json:"PublishAllPorts,omitempty"`
	// ReadonlyPaths is the list of paths to be set as read-only inside the container (this overrides the default set of paths)
	ReadonlyPaths []string `json:"ReadonlyPaths,omitempty"`
	ReadonlyRootfs bool `json:"ReadonlyRootfs,omitempty"`
	RestartPolicy *RestartPolicy `json:"RestartPolicy,omitempty"`
	Runtime string `json:"Runtime,omitempty"`
	SecurityOpt []string `json:"SecurityOpt,omitempty"`
	ShmSize int64 `json:"ShmSize,omitempty"`
	StorageOpt map[string]string `json:"StorageOpt,omitempty"`
	Sysctls map[string]string `json:"Sysctls,omitempty"`
	Tmpfs map[string]string `json:"Tmpfs,omitempty"`
	UTSMode *UtsMode `json:"UTSMode,omitempty"`
	Ulimits []Ulimit `json:"Ulimits,omitempty"`
	UsernsMode *UsernsMode `json:"UsernsMode,omitempty"`
	VolumeDriver string `json:"VolumeDriver,omitempty"`
	VolumesFrom []string `json:"VolumesFrom,omitempty"`
}
