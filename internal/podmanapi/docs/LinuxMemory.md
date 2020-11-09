# LinuxMemory

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**DisableOOMKiller** | **bool** | DisableOOMKiller disables the OOM killer for out of memory conditions | [optional] [default to null]
**Kernel** | **int64** | Kernel memory limit (in bytes). | [optional] [default to null]
**KernelTCP** | **int64** | Kernel memory limit for tcp (in bytes) | [optional] [default to null]
**Limit** | **int64** | Memory limit (in bytes). | [optional] [default to null]
**Reservation** | **int64** | Memory reservation or soft_limit (in bytes). | [optional] [default to null]
**Swap** | **int64** | Total memory limit (memory + swap). | [optional] [default to null]
**Swappiness** | **int32** | How aggressive the kernel will swap memory pages. | [optional] [default to null]
**UseHierarchy** | **bool** | Enables hierarchical memory accounting | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

