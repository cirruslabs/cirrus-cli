# NetworkCreate

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Attachable** | **bool** |  | [optional] [default to null]
**CheckDuplicate** | **bool** | Check for networks with duplicate names. Network is primarily keyed based on a random ID and not on the name. Network name is strictly a user-friendly alias to the network which is uniquely identified using ID. And there is no guaranteed way to check for duplicates. Option CheckDuplicate is there to provide a best effort checking of any networks which has the same name but it is not guaranteed to catch all name collisions. | [optional] [default to null]
**ConfigFrom** | [***ConfigReference**](ConfigReference.md) |  | [optional] [default to null]
**ConfigOnly** | **bool** |  | [optional] [default to null]
**Driver** | **string** |  | [optional] [default to null]
**EnableIPv6** | **bool** |  | [optional] [default to null]
**IPAM** | [***Ipam**](IPAM.md) |  | [optional] [default to null]
**Ingress** | **bool** |  | [optional] [default to null]
**Internal** | **bool** |  | [optional] [default to null]
**Labels** | **map[string]string** |  | [optional] [default to null]
**Options** | **map[string]string** |  | [optional] [default to null]
**Scope** | **string** |  | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

