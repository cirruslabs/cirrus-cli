# LibpodImagesRemoveReport

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Deleted** | **[]string** | Deleted images. | [optional] [default to null]
**Errors** | **[]string** | Image removal requires is to return data and an error. | [optional] [default to null]
**ExitCode** | **int64** | ExitCode describes the exit codes as described in the &#x60;podman rmi&#x60; man page. | [optional] [default to null]
**Untagged** | **[]string** | Untagged images. Can be longer than Deleted. | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

