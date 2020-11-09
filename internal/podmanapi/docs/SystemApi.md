# {{classname}}

All URIs are relative to *http://podman.io/*

Method | HTTP request | Description
------------- | ------------- | -------------
[**Df**](SystemApi.md#Df) | **Get** /libpod/system/df | Show disk usage
[**LibpodGetEvents**](SystemApi.md#LibpodGetEvents) | **Get** /libpod/events | Get events
[**LibpodGetInfo**](SystemApi.md#LibpodGetInfo) | **Get** /libpod/info | Get info
[**LibpodPingGet**](SystemApi.md#LibpodPingGet) | **Get** /libpod/_ping | Ping service
[**PruneSystem**](SystemApi.md#PruneSystem) | **Post** /libpod/system/prune | Prune unused data
[**SystemVersion**](SystemApi.md#SystemVersion) | **Get** /libpod/version | Component Version information

# **Df**
> InlineResponse20012 Df(ctx, )
Show disk usage

Return information about disk usage for containers, images, and volumes

### Required Parameters
This endpoint does not need any parameter.

### Return type

[**InlineResponse20012**](inline_response_200_12.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **LibpodGetEvents**
> LibpodGetEvents(ctx, optional)
Get events

Returns events filtered on query parameters

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***SystemApiLibpodGetEventsOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a SystemApiLibpodGetEventsOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **since** | **optional.String**| start streaming events from this time | 
 **until** | **optional.String**| stop streaming events later than this | 
 **filters** | **optional.String**| JSON encoded map[string][]string of constraints | 
 **stream** | **optional.Bool**| when false, do not follow events | [default to true]

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **LibpodGetInfo**
> Info LibpodGetInfo(ctx, )
Get info

Returns information on the system and libpod configuration

### Required Parameters
This endpoint does not need any parameter.

### Return type

[**Info**](Info.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **LibpodPingGet**
> string LibpodPingGet(ctx, )
Ping service

Return protocol information in response headers. `HEAD /libpod/_ping` is also supported. `/_ping` is available for compatibility with other engines. The '_ping' endpoints are not versioned. 

### Required Parameters
This endpoint does not need any parameter.

### Return type

**string**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: text/plain

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **PruneSystem**
> InlineResponse20013 PruneSystem(ctx, )
Prune unused data

### Required Parameters
This endpoint does not need any parameter.

### Return type

[**InlineResponse20013**](inline_response_200_13.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **SystemVersion**
> InlineResponse20014 SystemVersion(ctx, )
Component Version information

### Required Parameters
This endpoint does not need any parameter.

### Return type

[**InlineResponse20014**](inline_response_200_14.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

