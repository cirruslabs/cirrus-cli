# {{classname}}

All URIs are relative to *http://podman.io/*

Method | HTTP request | Description
------------- | ------------- | -------------
[**LibpodCreateExec**](ExecApi.md#LibpodCreateExec) | **Post** /libpod/containers/{name}/exec | Create an exec instance
[**LibpodInspectExec**](ExecApi.md#LibpodInspectExec) | **Get** /libpod/exec/{id}/json | Inspect an exec instance
[**LibpodResizeExec**](ExecApi.md#LibpodResizeExec) | **Post** /libpod/exec/{id}/resize | Resize an exec instance
[**LibpodStartExec**](ExecApi.md#LibpodStartExec) | **Post** /libpod/exec/{id}/start | Start an exec instance

# **LibpodCreateExec**
> LibpodCreateExec(ctx, name, optional)
Create an exec instance

Create an exec session to run a command inside a running container. Exec sessions will be automatically removed 5 minutes after they exit.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name** | **string**| name of container | 
 **optional** | ***ExecApiLibpodCreateExecOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a ExecApiLibpodCreateExecOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **body** | [**optional.Interface of Body4**](Body4.md)| Attributes for create | 

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json, application/x-tar
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **LibpodInspectExec**
> LibpodInspectExec(ctx, id)
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

# **LibpodResizeExec**
> LibpodResizeExec(ctx, id, optional)
Resize an exec instance

Resize the TTY session used by an exec instance. This endpoint only works if tty was specified as part of creating and starting the exec instance. 

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **id** | **string**| Exec instance ID | 
 **optional** | ***ExecApiLibpodResizeExecOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a ExecApiLibpodResizeExecOpts struct
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

# **LibpodStartExec**
> LibpodStartExec(ctx, id, optional)
Start an exec instance

Starts a previously set up exec instance. If detach is true, this endpoint returns immediately after starting the command. Otherwise, it sets up an interactive session with the command.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **id** | **string**| Exec instance ID | 
 **optional** | ***ExecApiLibpodStartExecOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a ExecApiLibpodStartExecOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **body** | [**optional.Interface of Body6**](Body6.md)| Attributes for start | 

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json, application/x-tar
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

