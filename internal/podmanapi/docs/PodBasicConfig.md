# PodBasicConfig

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Hostname** | **string** | Hostname is the pod&#x27;s hostname. If not set, the name of the pod will be used (if a name was not provided here, the name auto-generated for the pod will be used). This will be used by the infra container and all containers in the pod as long as the UTS namespace is shared. Optional. | [optional] [default to null]
**InfraCommand** | **[]string** | InfraCommand sets the command that will be used to start the infra container. If not set, the default set in the Libpod configuration file will be used. Conflicts with NoInfra&#x3D;true. Optional. | [optional] [default to null]
**InfraConmonPidFile** | **string** | InfraConmonPidFile is a custom path to store the infra container&#x27;s conmon PID. | [optional] [default to null]
**InfraImage** | **string** | InfraImage is the image that will be used for the infra container. If not set, the default set in the Libpod configuration file will be used. Conflicts with NoInfra&#x3D;true. Optional. | [optional] [default to null]
**Labels** | **map[string]string** | Labels are key-value pairs that are used to add metadata to pods. Optional. | [optional] [default to null]
**Name** | **string** | Name is the name of the pod. If not provided, a name will be generated when the pod is created. Optional. | [optional] [default to null]
**NoInfra** | **bool** | NoInfra tells the pod not to create an infra container. If this is done, many networking-related options will become unavailable. Conflicts with setting any options in PodNetworkConfig, and the InfraCommand and InfraImages in this struct. Optional. | [optional] [default to null]
**PodCreateCommand** | **[]string** | PodCreateCommand is the command used to create this pod. This will be shown in the output of Inspect() on the pod, and may also be used by some tools that wish to recreate the pod (e.g. &#x60;podman generate systemd --new&#x60;). Optional. | [optional] [default to null]
**SharedNamespaces** | **[]string** | SharedNamespaces instructs the pod to share a set of namespaces. Shared namespaces will be joined (by default) by every container which joins the pod. If not set and NoInfra is false, the pod will set a default set of namespaces to share. Conflicts with NoInfra&#x3D;true. Optional. | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

