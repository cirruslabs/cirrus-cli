# InlineResponse20015

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Anonymous** | **bool** | Anonymous indicates that the volume was created as an anonymous volume for a specific container, and will be be removed when any container using it is removed. | [optional] [default to null]
**CreatedAt** | [**time.Time**](time.Time.md) | CreatedAt is the date and time the volume was created at. This is not stored for older Libpod volumes; if so, it will be omitted. | [optional] [default to null]
**Driver** | **string** | Driver is the driver used to create the volume. This will be properly implemented in a future version. | [optional] [default to null]
**GID** | **int64** | GID is the GID that the volume was created with. | [optional] [default to null]
**Labels** | **map[string]string** | Labels includes the volume&#x27;s configured labels, key:value pairs that can be passed during volume creation to provide information for third party tools. | [optional] [default to null]
**Mountpoint** | **string** | Mountpoint is the path on the host where the volume is mounted. | [optional] [default to null]
**Name** | **string** | Name is the name of the volume. | [optional] [default to null]
**Options** | **map[string]string** | Options is a set of options that were used when creating the volume. It is presently not used. | [optional] [default to null]
**Scope** | **string** | Scope is unused and provided solely for Docker compatibility. It is unconditionally set to \&quot;local\&quot;. | [optional] [default to null]
**Status** | **map[string]string** | Status is presently unused and provided only for Docker compatibility. In the future it will be used to return information on the volume&#x27;s current state. | [optional] [default to null]
**UID** | **int64** | UID is the UID that the volume was created with. | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

