# {{classname}}

All URIs are relative to *http://podman.io/*

Method | HTTP request | Description
------------- | ------------- | -------------
[**AddManifest**](ManifestsApi.md#AddManifest) | **Post** /libpod/manifests/{name:.*}/add | 
[**Create**](ManifestsApi.md#Create) | **Post** /libpod/manifests/create | Create
[**Inspect**](ManifestsApi.md#Inspect) | **Get** /libpod/manifests/{name:.*}/json | Inspect
[**PushManifest**](ManifestsApi.md#PushManifest) | **Post** /libpod/manifests/{name}/push | Push
[**RemoveManifest**](ManifestsApi.md#RemoveManifest) | **Delete** /libpod/manifests/{name:.*} | Remove

# **AddManifest**
> AddManifest(ctx, name_, optional)


Add an image to a manifest list

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name_** | **string**| the name or ID of the manifest | 
 **optional** | ***ManifestsApiAddManifestOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a ManifestsApiAddManifestOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **body** | [**optional.Interface of ManifestAddOpts**](ManifestAddOpts.md)| options for creating a manifest | 

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json, application/x-tar
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **Create**
> Create(ctx, name, optional)
Create

Create a manifest list

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name** | **string**| manifest list name | 
 **optional** | ***ManifestsApiCreateOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a ManifestsApiCreateOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **image** | **optional.String**| name of the image | 
 **all** | **optional.Bool**| add all contents if given list | 

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **Inspect**
> List Inspect(ctx, name_)
Inspect

Display a manifest list

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name_** | **string**| the name or ID of the manifest | 

### Return type

[**List**](List.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **PushManifest**
> PushManifest(ctx, name, destination, optional)
Push

Push a manifest list or image index to a registry

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name** | **string**| the name or ID of the manifest | 
  **destination** | **string**| the destination for the manifest | 
 **optional** | ***ManifestsApiPushManifestOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a ManifestsApiPushManifestOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


 **all** | **optional.Bool**| push all images | 

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **RemoveManifest**
> RemoveManifest(ctx, name_, optional)
Remove

Remove an image from a manifest list

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name_** | **string**| the image associated with the manifest | 
 **optional** | ***ManifestsApiRemoveManifestOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a ManifestsApiRemoveManifestOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **digest** | **optional.String**| image digest to be removed | 

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

