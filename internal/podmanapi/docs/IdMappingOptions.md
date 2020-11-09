# IdMappingOptions

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**AutoUserNs** | **bool** |  | [optional] [default to null]
**AutoUserNsOpts** | [***AutoUserNsOptions**](AutoUserNsOptions.md) |  | [optional] [default to null]
**GIDMap** | [**[]IdMap**](IDMap.md) |  | [optional] [default to null]
**HostGIDMapping** | **bool** |  | [optional] [default to null]
**HostUIDMapping** | **bool** | UIDMap and GIDMap are used for setting up a layer&#x27;s root filesystem for use inside of a user namespace where ID mapping is being used. If HostUIDMapping/HostGIDMapping is true, no mapping of the respective type will be used.  Otherwise, if UIDMap and/or GIDMap contain at least one mapping, one or both will be used.  By default, if neither of those conditions apply, if the layer has a parent layer, the parent layer&#x27;s mapping will be used, and if it does not have a parent layer, the mapping which was passed to the Store object when it was initialized will be used. | [optional] [default to null]
**UIDMap** | [**[]IdMap**](IDMap.md) |  | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

