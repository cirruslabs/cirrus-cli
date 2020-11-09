# {{classname}}

All URIs are relative to *http://podman.io/*

Method | HTTP request | Description
------------- | ------------- | -------------
[**LibpodCreateVolume**](VolumesApi.md#LibpodCreateVolume) | **Post** /libpod/volumes/create | Create a volume
[**LibpodInspectVolume**](VolumesApi.md#LibpodInspectVolume) | **Get** /libpod/volumes/{name}/json | Inspect volume
[**LibpodListVolumes**](VolumesApi.md#LibpodListVolumes) | **Get** /libpod/volumes/json | List volumes
[**LibpodPruneVolumes**](VolumesApi.md#LibpodPruneVolumes) | **Post** /libpod/volumes/prune | Prune volumes
[**LibpodRemoveVolume**](VolumesApi.md#LibpodRemoveVolume) | **Delete** /libpod/volumes/{name} | Remove volume

# **LibpodCreateVolume**
> InlineResponse20015 LibpodCreateVolume(ctx, optional)
Create a volume

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***VolumesApiLibpodCreateVolumeOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a VolumesApiLibpodCreateVolumeOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**optional.Interface of VolumeCreate**](VolumeCreate.md)| attributes for creating a container | 

### Return type

[**InlineResponse20015**](inline_response_200_15.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json, application/x-tar
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **LibpodInspectVolume**
> InlineResponse20015 LibpodInspectVolume(ctx, name)
Inspect volume

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name** | **string**| the name or ID of the volume | 

### Return type

[**InlineResponse20015**](inline_response_200_15.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **LibpodListVolumes**
> []Volume LibpodListVolumes(ctx, optional)
List volumes

Returns a list of volumes

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***VolumesApiLibpodListVolumesOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a VolumesApiLibpodListVolumesOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **filters** | **optional.String**| JSON encoded value of the filters (a map[string][]string) to process on the volumes list. Available filters:   - driver&#x3D;&lt;volume-driver-name&gt; Matches volumes based on their driver.   - label&#x3D;&lt;key&gt; or label&#x3D;&lt;key&gt;:&lt;value&gt; Matches volumes based on the presence of a label alone or a label and a value.   - name&#x3D;&lt;volume-name&gt; Matches all of volume name.   - opt&#x3D;&lt;driver-option&gt; Matches a storage driver options  | 

### Return type

[**[]Volume**](Volume.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **LibpodPruneVolumes**
> []VolumePruneReport LibpodPruneVolumes(ctx, )
Prune volumes

### Required Parameters
This endpoint does not need any parameter.

### Return type

[**[]VolumePruneReport**](VolumePruneReport.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **LibpodRemoveVolume**
> LibpodRemoveVolume(ctx, name, optional)
Remove volume

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name** | **string**| the name or ID of the volume | 
 **optional** | ***VolumesApiLibpodRemoveVolumeOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a VolumesApiLibpodRemoveVolumeOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **force** | **optional.Bool**| force removal | 

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

