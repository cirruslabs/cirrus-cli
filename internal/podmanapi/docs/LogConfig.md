# LogConfig

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Driver** | **string** | LogDriver is the container&#x27;s log driver. Optional. | [optional] [default to null]
**Options** | **map[string]string** | A set of options to accompany the log driver. Optional. | [optional] [default to null]
**Path** | **string** | LogPath is the path the container&#x27;s logs will be stored at. Only available if LogDriver is set to \&quot;json-file\&quot; or \&quot;k8s-file\&quot;. Optional. | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

