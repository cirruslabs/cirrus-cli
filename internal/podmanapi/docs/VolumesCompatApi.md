# {{classname}}

All URIs are relative to *http://podman.io/*

Method | HTTP request | Description
------------- | ------------- | -------------
[**CreateVolume**](VolumesCompatApi.md#CreateVolume) | **Post** /volumes/create | Create a volume
[**InspectVolume**](VolumesCompatApi.md#InspectVolume) | **Get** /volumes/{name} | Inspect volume
[**ListVolumes**](VolumesCompatApi.md#ListVolumes) | **Get** /volumes | List volumes
[**PruneVolumes**](VolumesCompatApi.md#PruneVolumes) | **Post** /volumes/prune | Prune volumes
[**RemoveVolume**](VolumesCompatApi.md#RemoveVolume) | **Delete** /volumes/{name} | Remove volume

# **CreateVolume**
> InlineResponse20018 CreateVolume(ctx, optional)
Create a volume

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***VolumesCompatApiCreateVolumeOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a VolumesCompatApiCreateVolumeOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**optional.Interface of VolumeCreateBody**](VolumeCreateBody.md)| attributes for creating a container.
Note: If a volume by the same name exists, a 201 response with that volume&#x27;s information will be generated.
 | 

### Return type

[**InlineResponse20018**](inline_response_200_18.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json, application/x-tar
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **InspectVolume**
> InlineResponse20018 InspectVolume(ctx, name)
Inspect volume

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name** | **string**| the name or ID of the volume | 

### Return type

[**InlineResponse20018**](inline_response_200_18.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ListVolumes**
> InlineResponse20017 ListVolumes(ctx, optional)
List volumes

Returns a list of volume

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***VolumesCompatApiListVolumesOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a VolumesCompatApiListVolumesOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **filters** | **optional.String**| JSON encoded value of the filters (a map[string][]string) to process on the volumes list. Available filters:   - driver&#x3D;&lt;volume-driver-name&gt; Matches volumes based on their driver.   - label&#x3D;&lt;key&gt; or label&#x3D;&lt;key&gt;:&lt;value&gt; Matches volumes based on the presence of a label alone or a label and a value.   - name&#x3D;&lt;volume-name&gt; Matches all of volume name.  Note:   The boolean &#x60;dangling&#x60; filter is not yet implemented for this endpoint.  | 

### Return type

[**InlineResponse20017**](inline_response_200_17.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **PruneVolumes**
> InlineResponse20019 PruneVolumes(ctx, optional)
Prune volumes

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***VolumesCompatApiPruneVolumesOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a VolumesCompatApiPruneVolumesOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **filters** | **optional.String**| JSON encoded value of filters (a map[string][]string) to match volumes against before pruning.  Note: No filters are currently supported and any filters specified will cause an error response.  | 

### Return type

[**InlineResponse20019**](inline_response_200_19.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **RemoveVolume**
> RemoveVolume(ctx, name, optional)
Remove volume

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name** | **string**| the name or ID of the volume | 
 **optional** | ***VolumesCompatApiRemoveVolumeOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a VolumesCompatApiRemoveVolumeOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **force** | **optional.Bool**| Force removal of the volume. This actually only causes errors due to the names volume not being found to be suppressed, which is the behaviour Docker implements.  | 

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

