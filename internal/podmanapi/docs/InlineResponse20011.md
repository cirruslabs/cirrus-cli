# InlineResponse20011

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**CgroupParent** | **string** | CgroupParent is the parent of the pod&#x27;s CGroup. | [optional] [default to null]
**CgroupPath** | **string** | CgroupPath is the path to the pod&#x27;s CGroup. | [optional] [default to null]
**Containers** | [**[]InspectPodContainerInfo**](InspectPodContainerInfo.md) | Containers gives a brief summary of all containers in the pod and their current status. | [optional] [default to null]
**CreateCgroup** | **bool** | CreateCgroup is whether this pod will create its own CGroup to group containers under. | [optional] [default to null]
**CreateCommand** | **[]string** | CreateCommand is the full command plus arguments of the process the container has been created with. | [optional] [default to null]
**CreateInfra** | **bool** | CreateInfra is whether this pod will create an infra container to share namespaces. | [optional] [default to null]
**Created** | [**time.Time**](time.Time.md) | Created is the time when the pod was created. | [optional] [default to null]
**Hostname** | **string** | Hostname is the hostname that the pod will set. | [optional] [default to null]
**Id** | **string** | ID is the ID of the pod. | [optional] [default to null]
**InfraConfig** | [***InspectPodInfraConfig**](InspectPodInfraConfig.md) |  | [optional] [default to null]
**InfraContainerID** | **string** | InfraContainerID is the ID of the pod&#x27;s infra container, if one is present. | [optional] [default to null]
**Labels** | **map[string]string** | Labels is a set of key-value labels that have been applied to the pod. | [optional] [default to null]
**Name** | **string** | Name is the name of the pod. | [optional] [default to null]
**Namespace** | **string** | Namespace is the Libpod namespace the pod is placed in. | [optional] [default to null]
**NumContainers** | **int32** | NumContainers is the number of containers in the pod, including the infra container. | [optional] [default to null]
**SharedNamespaces** | **[]string** | SharedNamespaces contains a list of namespaces that will be shared by containers within the pod. Can only be set if CreateInfra is true. | [optional] [default to null]
**State** | **string** | State represents the current state of the pod. | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

