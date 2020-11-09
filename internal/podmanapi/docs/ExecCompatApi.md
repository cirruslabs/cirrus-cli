# {{classname}}

All URIs are relative to *http://podman.io/*

Method | HTTP request | Description
------------- | ------------- | -------------
[**CreateExec**](ExecCompatApi.md#CreateExec) | **Post** /containers/{name}/exec | Create an exec instance
[**InspectExec**](ExecCompatApi.md#InspectExec) | **Get** /exec/{id}/json | Inspect an exec instance
[**ResizeExec**](ExecCompatApi.md#ResizeExec) | **Post** /exec/{id}/resize | Resize an exec instance
[**StartExec**](ExecCompatApi.md#StartExec) | **Post** /exec/{id}/start | Start an exec instance

# **CreateExec**
> CreateExec(ctx, name, optional)
Create an exec instance

Create an exec session to run a command inside a running container. Exec sessions will be automatically removed 5 minutes after they exit.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name** | **string**| name of container | 
 **optional** | ***ExecCompatApiCreateExecOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a ExecCompatApiCreateExecOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **body** | [**optional.Interface of Body**](Body.md)| Attributes for create | 

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json, application/x-tar
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **InspectExec**
> InspectExec(ctx, id)
Inspect an exec instance

Return low-level information about an exec instance.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **id** | **string**| Exec instance ID | 

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ResizeExec**
> ResizeExec(ctx, id, optional)
Resize an exec instance

Resize the TTY session used by an exec instance. This endpoint only works if tty was specified as part of creating and starting the exec instance. 

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **id** | **string**| Exec instance ID | 
 **optional** | ***ExecCompatApiResizeExecOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a ExecCompatApiResizeExecOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **h** | **optional.Int32**| Height of the TTY session in characters | 
 **w** | **optional.Int32**| Width of the TTY session in characters | 

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **StartExec**
> StartExec(ctx, id, optional)
Start an exec instance

Starts a previously set up exec instance. If detach is true, this endpoint returns immediately after starting the command. Otherwise, it sets up an interactive session with the command.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **id** | **string**| Exec instance ID | 
 **optional** | ***ExecCompatApiStartExecOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a ExecCompatApiStartExecOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **body** | [**optional.Interface of Body2**](Body2.md)| Attributes for start | 

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json, application/x-tar
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

