# ContainerCgroupConfig

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**CgroupParent** | **string** | CgroupParent is the container&#x27;s CGroup parent. If not set, the default for the current cgroup driver will be used. Optional. | [optional] [default to null]
**Cgroupns** | [***Namespace**](Namespace.md) |  | [optional] [default to null]
**CgroupsMode** | **string** | CgroupsMode sets a policy for how cgroups will be created in the container, including the ability to disable creation entirely. | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

