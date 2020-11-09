# {{classname}}

All URIs are relative to *http://podman.io/*

Method | HTTP request | Description
------------- | ------------- | -------------
[**LibpodCreateNetwork**](NetworksApi.md#LibpodCreateNetwork) | **Post** /libpod/networks/create | Create network
[**LibpodInspectNetwork**](NetworksApi.md#LibpodInspectNetwork) | **Get** /libpod/networks/{name}/json | Inspect a network
[**LibpodListNetwork**](NetworksApi.md#LibpodListNetwork) | **Get** /libpod/networks/json | List networks
[**LibpodRemoveNetwork**](NetworksApi.md#LibpodRemoveNetwork) | **Delete** /libpod/networks/{name} | Remove a network

# **LibpodCreateNetwork**
> NetworkCreateReport LibpodCreateNetwork(ctx, optional)
Create network

Create a new CNI network configuration

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***NetworksApiLibpodCreateNetworkOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a NetworksApiLibpodCreateNetworkOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**optional.Interface of NetworkCreateOptions**](NetworkCreateOptions.md)| attributes for creating a container | 
 **name** | **optional.**| optional name for new network | 

### Return type

[**NetworkCreateReport**](NetworkCreateReport.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json, application/x-tar
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **LibpodInspectNetwork**
> []map[string]interface{} LibpodInspectNetwork(ctx, name)
Inspect a network

Display low level configuration for a CNI network

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name** | **string**| the name of the network | 

### Return type

[**[]map[string]interface{}**](map.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **LibpodListNetwork**
> []NetworkListReport LibpodListNetwork(ctx, optional)
List networks

Display summary of network configurations

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***NetworksApiLibpodListNetworkOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a NetworksApiLibpodListNetworkOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **filter** | **optional.String**| Provide filter values (e.g. &#x27;name&#x3D;podman&#x27;) | 

### Return type

[**[]NetworkListReport**](NetworkListReport.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **LibpodRemoveNetwork**
> NetworkRmReport LibpodRemoveNetwork(ctx, name, optional)
Remove a network

Remove a CNI configured network

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name** | **string**| the name of the network | 
 **optional** | ***NetworksApiLibpodRemoveNetworkOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a NetworksApiLibpodRemoveNetworkOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **force** | **optional.Bool**| remove containers associated with network | 

### Return type

[**NetworkRmReport**](NetworkRmReport.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

