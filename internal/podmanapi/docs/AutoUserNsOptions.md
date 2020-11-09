# AutoUserNsOptions

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**AdditionalGIDMappings** | [**[]IdMap**](IDMap.md) | AdditionalGIDMappings specified additional GID mappings to include in the generated user namespace. | [optional] [default to null]
**AdditionalUIDMappings** | [**[]IdMap**](IDMap.md) | AdditionalUIDMappings specified additional UID mappings to include in the generated user namespace. | [optional] [default to null]
**GroupFile** | **string** | GroupFile to use if the container uses a volume. | [optional] [default to null]
**InitialSize** | **int32** | InitialSize defines the minimum size for the user namespace. The created user namespace will have at least this size. | [optional] [default to null]
**PasswdFile** | **string** | PasswdFile to use if the container uses a volume. | [optional] [default to null]
**Size** | **int32** | Size defines the size for the user namespace.  If it is set to a value bigger than 0, the user namespace will have exactly this size. If it is not set, some heuristics will be used to find its size. | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

