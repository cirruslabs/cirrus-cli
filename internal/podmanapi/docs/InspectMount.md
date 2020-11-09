# InspectMount

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Destination** | **string** | The destination directory for the volume. Specified as a path within the container, as it would be passed into the OCI runtime. | [optional] [default to null]
**Driver** | **string** | The driver used for the named volume. Empty for bind mounts. | [optional] [default to null]
**Mode** | **string** | Contains SELinux :z/:Z mount options. Unclear what, if anything, else goes in here. | [optional] [default to null]
**Name** | **string** | The name of the volume. Empty for bind mounts. | [optional] [default to null]
**Options** | **[]string** | All remaining mount options. Additional data, not present in the original output. | [optional] [default to null]
**Propagation** | **string** | Mount propagation for the mount. Can be empty if not specified, but is always printed - no omitempty. | [optional] [default to null]
**RW** | **bool** | Whether the volume is read-write | [optional] [default to null]
**Source** | **string** | The source directory for the volume. | [optional] [default to null]
**Type_** | **string** | Whether the mount is a volume or bind mount. Allowed values are \&quot;volume\&quot; and \&quot;bind\&quot;. | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

