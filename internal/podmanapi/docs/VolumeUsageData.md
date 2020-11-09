# VolumeUsageData

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**RefCount** | **int64** | The number of containers referencing this volume. This field is set to &#x60;-1&#x60; if the reference-count is not available. | [default to null]
**Size** | **int64** | Amount of disk space used by the volume (in bytes). This information is only available for volumes created with the &#x60;\&quot;local\&quot;&#x60; volume driver. For volumes created with other volume drivers, this field is set to &#x60;-1&#x60; (\&quot;not available\&quot;) | [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

