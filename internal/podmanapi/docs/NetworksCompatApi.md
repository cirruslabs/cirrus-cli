# {{classname}}

All URIs are relative to *http://podman.io/*

Method | HTTP request | Description
------------- | ------------- | -------------
[**CompatCreateNetwork**](NetworksCompatApi.md#CompatCreateNetwork) | **Post** /networks/create | Create network
[**CompatInspectNetwork**](NetworksCompatApi.md#CompatInspectNetwork) | **Get** /networks/{name} | Inspect a network
[**CompatListNetwork**](NetworksCompatApi.md#CompatListNetwork) | **Get** /networks | List networks
[**CompatRemoveNetwork**](NetworksCompatApi.md#CompatRemoveNetwork) | **Delete** /networks/{name} | Remove a network

# **CompatCreateNetwork**
> InlineResponse20016 CompatCreateNetwork(ctx, optional)
Create network

Create a network configuration

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***NetworksCompatApiCompatCreateNetworkOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a NetworksCompatApiCompatCreateNetworkOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**optional.Interface of NetworkCreateRequest**](NetworkCreateRequest.md)| attributes for creating a container | 

### Return type

[**InlineResponse20016**](inline_response_200_16.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json, application/x-tar
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **CompatInspectNetwork**
> NetworkResource CompatInspectNetwork(ctx, name)
Inspect a network

Display low level configuration network

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name** | **string**| the name of the network | 

### Return type

[**NetworkResource**](NetworkResource.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **CompatListNetwork**
> []NetworkResource CompatListNetwork(ctx, optional)
List networks

Display summary of network configurations

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***NetworksCompatApiCompatListNetworkOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a NetworksCompatApiCompatListNetworkOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **filters** | **optional.String**| JSON encoded value of the filters (a map[string][]string) to process on the networks list. Only the name filter is supported. | 

### Return type

[**[]NetworkResource**](NetworkResource.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **CompatRemoveNetwork**
> CompatRemoveNetwork(ctx, name)
Remove a network

Remove a network

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name** | **string**| the name of the network | 

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

