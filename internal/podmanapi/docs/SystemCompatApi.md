# {{classname}}

All URIs are relative to *http://podman.io/*

Method | HTTP request | Description
------------- | ------------- | -------------
[**CompatSystemVersion**](SystemCompatApi.md#CompatSystemVersion) | **Get** /version | Component Version information
[**GetEvents**](SystemCompatApi.md#GetEvents) | **Get** /events | Get events
[**GetInfo**](SystemCompatApi.md#GetInfo) | **Get** /info | Get info
[**LibpodPingGet**](SystemCompatApi.md#LibpodPingGet) | **Get** /libpod/_ping | Ping service

# **CompatSystemVersion**
> InlineResponse20014 CompatSystemVersion(ctx, )
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

# **GetEvents**
> GetEvents(ctx, optional)
Get events

Returns events filtered on query parameters

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***SystemCompatApiGetEventsOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a SystemCompatApiGetEventsOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **since** | **optional.String**| start streaming events from this time | 
 **until** | **optional.String**| stop streaming events later than this | 
 **filters** | **optional.String**| JSON encoded map[string][]string of constraints | 

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetInfo**
> GetInfo(ctx, )
Get info

Returns information on the system and libpod configuration

### Required Parameters
This endpoint does not need any parameter.

### Return type

 (empty response body)

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

