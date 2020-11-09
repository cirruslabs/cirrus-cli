# LinuxResources

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**BlockIO** | [***LinuxBlockIo**](LinuxBlockIO.md) |  | [optional] [default to null]
**Cpu** | [***LinuxCpu**](LinuxCPU.md) |  | [optional] [default to null]
**Devices** | [**[]LinuxDeviceCgroup**](LinuxDeviceCgroup.md) | Devices configures the device allowlist. | [optional] [default to null]
**HugepageLimits** | [**[]LinuxHugepageLimit**](LinuxHugepageLimit.md) | Hugetlb limit (in bytes) | [optional] [default to null]
**Memory** | [***LinuxMemory**](LinuxMemory.md) |  | [optional] [default to null]
**Network** | [***LinuxNetwork**](LinuxNetwork.md) |  | [optional] [default to null]
**Pids** | [***LinuxPids**](LinuxPids.md) |  | [optional] [default to null]
**Rdma** | [**map[string]LinuxRdma**](LinuxRdma.md) | Rdma resource restriction configuration. Limits are a set of key value pairs that define RDMA resource limits, where the key is device name and value is resource limits. | [optional] [default to null]
**Unified** | **map[string]string** | Unified resources. | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

