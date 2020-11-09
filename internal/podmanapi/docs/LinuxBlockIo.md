# LinuxBlockIo

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**LeafWeight** | **int32** | Specifies tasks&#x27; weight in the given cgroup while competing with the cgroup&#x27;s child cgroups, CFQ scheduler only | [optional] [default to null]
**ThrottleReadBpsDevice** | [**[]LinuxThrottleDevice**](LinuxThrottleDevice.md) | IO read rate limit per cgroup per device, bytes per second | [optional] [default to null]
**ThrottleReadIOPSDevice** | [**[]LinuxThrottleDevice**](LinuxThrottleDevice.md) | IO read rate limit per cgroup per device, IO per second | [optional] [default to null]
**ThrottleWriteBpsDevice** | [**[]LinuxThrottleDevice**](LinuxThrottleDevice.md) | IO write rate limit per cgroup per device, bytes per second | [optional] [default to null]
**ThrottleWriteIOPSDevice** | [**[]LinuxThrottleDevice**](LinuxThrottleDevice.md) | IO write rate limit per cgroup per device, IO per second | [optional] [default to null]
**Weight** | **int32** | Specifies per cgroup weight | [optional] [default to null]
**WeightDevice** | [**[]LinuxWeightDevice**](LinuxWeightDevice.md) | Weight per cgroup per device, can override BlkioWeight | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

