# NamedVolume

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Dest** | **string** | Destination to mount the named volume within the container. Must be an absolute path. Path will be created if it does not exist. | [optional] [default to null]
**Name** | **string** | Name is the name of the named volume to be mounted. May be empty. If empty, a new named volume with a pseudorandomly generated name will be mounted at the given destination. | [optional] [default to null]
**Options** | **[]string** | Options are options that the named volume will be mounted with. | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

