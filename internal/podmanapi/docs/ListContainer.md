# ListContainer

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Command** | **[]string** | Container command | [optional] [default to null]
**Created** | **int64** | Container creation time | [optional] [default to null]
**CreatedAt** | **string** | Human readable container creation time. | [optional] [default to null]
**ExitCode** | **int32** | If container has exited, the return code from the command | [optional] [default to null]
**Exited** | **bool** | If container has exited/stopped | [optional] [default to null]
**ExitedAt** | **int64** | Time container exited | [optional] [default to null]
**Id** | **string** | The unique identifier for the container | [optional] [default to null]
**Image** | **string** | Container image | [optional] [default to null]
**ImageID** | **string** | Container image ID | [optional] [default to null]
**IsInfra** | **bool** | If this container is a Pod infra container | [optional] [default to null]
**Labels** | **map[string]string** | Labels for container | [optional] [default to null]
**Mounts** | **[]string** | User volume mounts | [optional] [default to null]
**Names** | **[]string** | The names assigned to the container | [optional] [default to null]
**Namespaces** | [***ListContainerNamespaces**](ListContainerNamespaces.md) |  | [optional] [default to null]
**Pid** | **int64** | The process id of the container | [optional] [default to null]
**Pod** | **string** | If the container is part of Pod, the Pod ID. Requires the pod boolean to be set | [optional] [default to null]
**PodName** | **string** | If the container is part of Pod, the Pod name. Requires the pod boolean to be set | [optional] [default to null]
**Ports** | [**[]PortMapping**](PortMapping.md) | Port mappings | [optional] [default to null]
**Size** | [***ContainerSize**](ContainerSize.md) |  | [optional] [default to null]
**StartedAt** | **int64** | Time when container started | [optional] [default to null]
**State** | **string** | State of container | [optional] [default to null]
**Status** | **string** | Status is a human-readable approximation of a duration for json output | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

