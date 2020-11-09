# VolumeInfo

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**CreatedAt** | **string** | Date/Time the volume was created. | [optional] [default to null]
**Driver** | **string** | Name of the volume driver used by the volume. Only supports local driver | [default to null]
**Labels** | **map[string]string** | User-defined key/value metadata. Always included | [optional] [default to null]
**Mountpoint** | **string** | Mount path of the volume on the host. | [default to null]
**Name** | **string** | Name of the volume. | [default to null]
**Options** | **map[string]string** | The driver specific options used when creating the volume. | [default to null]
**Scope** | **string** | The level at which the volume exists. Libpod does not implement volume scoping, and this is provided solely for Docker compatibility. The value is only \&quot;local\&quot;. | [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

