# ContainerResourceConfig

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**OomScoreAdj** | **int64** | OOMScoreAdj adjusts the score used by the OOM killer to determine processes to kill for the container&#x27;s process. Optional. | [optional] [default to null]
**RLimits** | [**[]PosixRlimit**](POSIXRlimit.md) | Rlimits are POSIX rlimits to apply to the container. Optional. | [optional] [default to null]
**ResourceLimits** | [***LinuxResources**](LinuxResources.md) |  | [optional] [default to null]
**ThrottleReadBpsDevice** | [**map[string]LinuxThrottleDevice**](LinuxThrottleDevice.md) | IO read rate limit per cgroup per device, bytes per second | [optional] [default to null]
**ThrottleReadIOPSDevice** | [**map[string]LinuxThrottleDevice**](LinuxThrottleDevice.md) | IO read rate limit per cgroup per device, IO per second | [optional] [default to null]
**ThrottleWriteBpsDevice** | [**map[string]LinuxThrottleDevice**](LinuxThrottleDevice.md) | IO write rate limit per cgroup per device, bytes per second | [optional] [default to null]
**ThrottleWriteIOPSDevice** | [**map[string]LinuxThrottleDevice**](LinuxThrottleDevice.md) | IO write rate limit per cgroup per device, IO per second | [optional] [default to null]
**Unified** | **map[string]string** | CgroupConf are key-value options passed into the container runtime that are used to configure cgroup v2. Optional. | [optional] [default to null]
**WeightDevice** | [**map[string]LinuxWeightDevice**](LinuxWeightDevice.md) | Weight per cgroup per device, can override BlkioWeight | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

