# {{classname}}

All URIs are relative to *http://podman.io/*

Method | HTTP request | Description
------------- | ------------- | -------------
[**CreatePod**](PodsApi.md#CreatePod) | **Post** /libpod/pods/create | Create a pod
[**InspectPod**](PodsApi.md#InspectPod) | **Get** /libpod/pods/{name}/json | Inspect pod
[**KillPod**](PodsApi.md#KillPod) | **Post** /libpod/pods/{name}/kill | Kill a pod
[**LibpodGenerateKube**](PodsApi.md#LibpodGenerateKube) | **Get** /libpod/generate/{name:.*}/kube | Generate a Kubernetes YAML file.
[**LibpodGenerateSystemd**](PodsApi.md#LibpodGenerateSystemd) | **Get** /libpod/generate/{name:.*}/systemd | Generate Systemd Units
[**LibpodPlayKube**](PodsApi.md#LibpodPlayKube) | **Post** /libpod/play/kube | Play a Kubernetes YAML file.
[**ListPods**](PodsApi.md#ListPods) | **Get** /libpod/pods/json | List pods
[**PausePod**](PodsApi.md#PausePod) | **Post** /libpod/pods/{name}/pause | Pause a pod
[**PodExists**](PodsApi.md#PodExists) | **Get** /libpod/pods/{name}/exists | Pod exists
[**PrunePods**](PodsApi.md#PrunePods) | **Post** /libpod/pods/prune | Prune unused pods
[**RemovePod**](PodsApi.md#RemovePod) | **Delete** /libpod/pods/{name} | Remove pod
[**RestartPod**](PodsApi.md#RestartPod) | **Post** /libpod/pods/{name}/restart | Restart a pod
[**StartPod**](PodsApi.md#StartPod) | **Post** /libpod/pods/{name}/start | Start a pod
[**StatsPod**](PodsApi.md#StatsPod) | **Get** /libpod/pods/stats | Get stats for one or more pods
[**StopPod**](PodsApi.md#StopPod) | **Post** /libpod/pods/{name}/stop | Stop a pod
[**TopPod**](PodsApi.md#TopPod) | **Get** /libpod/pods/{name}/top | List processes
[**UnpausePod**](PodsApi.md#UnpausePod) | **Post** /libpod/pods/{name}/unpause | Unpause a pod

# **CreatePod**
> CreatePod(ctx, optional)
Create a pod

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***PodsApiCreatePodOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a PodsApiCreatePodOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**optional.Interface of PodSpecGenerator**](PodSpecGenerator.md)| attributes for creating a pod | 

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json, application/x-tar
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **InspectPod**
> InlineResponse20011 InspectPod(ctx, name)
Inspect pod

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name** | **string**| the name or ID of the pod | 

### Return type

[**InlineResponse20011**](inline_response_200_11.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **KillPod**
> PodKillReport KillPod(ctx, name, optional)
Kill a pod

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name** | **string**| the name or ID of the pod | 
 **optional** | ***PodsApiKillPodOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a PodsApiKillPodOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **signal** | **optional.String**| signal to be sent to pod | [default to SIGKILL]

### Return type

[**PodKillReport**](PodKillReport.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **LibpodGenerateKube**
> *os.File LibpodGenerateKube(ctx, name_, optional)
Generate a Kubernetes YAML file.

Generate Kubernetes YAML based on a pod or container.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name_** | **string**| Name or ID of the container or pod. | 
 **optional** | ***PodsApiLibpodGenerateKubeOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a PodsApiLibpodGenerateKubeOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **service** | **optional.Bool**| Generate YAML for a Kubernetes service object. | [default to false]

### Return type

[***os.File**](*os.File.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **LibpodGenerateSystemd**
> map[string]string LibpodGenerateSystemd(ctx, name_, optional)
Generate Systemd Units

Generate Systemd Units based on a pod or container.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name_** | **string**| Name or ID of the container or pod. | 
 **optional** | ***PodsApiLibpodGenerateSystemdOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a PodsApiLibpodGenerateSystemdOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **useName** | **optional.Bool**| Use container/pod names instead of IDs. | [default to false]
 **new** | **optional.Bool**| Create a new container instead of starting an existing one. | [default to false]
 **time** | **optional.Int32**| Stop timeout override. | [default to 10]
 **restartPolicy** | **optional.String**| Systemd restart-policy. | [default to on-failure]
 **containerPrefix** | **optional.String**| Systemd unit name prefix for containers. | [default to container]
 **podPrefix** | **optional.String**| Systemd unit name prefix for pods. | [default to pod]
 **separator** | **optional.String**| Systemd unit name separator between name/id and prefix. | [default to -]

### Return type

**map[string]string**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **LibpodPlayKube**
> PlayKubeReport LibpodPlayKube(ctx, optional)
Play a Kubernetes YAML file.

Create and run pods based on a Kubernetes YAML file (pod or service kind).

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***PodsApiLibpodPlayKubeOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a PodsApiLibpodPlayKubeOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**optional.Interface of string**](string.md)| Kubernetes YAML file. | 
 **network** | **optional.**| Connect the pod to this network. | 
 **tlsVerify** | **optional.**| Require HTTPS and verify signatures when contacting registries. | [default to true]

### Return type

[**PlayKubeReport**](PlayKubeReport.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json, application/x-tar
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ListPods**
> []ListPodsReport ListPods(ctx, optional)
List pods

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***PodsApiListPodsOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a PodsApiListPodsOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **filters** | **optional.String**| needs description and plumbing for filters | 

### Return type

[**[]ListPodsReport**](ListPodsReport.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **PausePod**
> PodPauseReport PausePod(ctx, name)
Pause a pod

Pause a pod

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name** | **string**| the name or ID of the pod | 

### Return type

[**PodPauseReport**](PodPauseReport.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **PodExists**
> PodExists(ctx, name)
Pod exists

Check if a pod exists by name or ID

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name** | **string**| the name or ID of the pod | 

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **PrunePods**
> PodPruneReport PrunePods(ctx, )
Prune unused pods

### Required Parameters
This endpoint does not need any parameter.

### Return type

[**PodPruneReport**](PodPruneReport.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **RemovePod**
> PodRmReport RemovePod(ctx, name, optional)
Remove pod

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name** | **string**| the name or ID of the pod | 
 **optional** | ***PodsApiRemovePodOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a PodsApiRemovePodOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **force** | **optional.Bool**| force removal of a running pod by first stopping all containers, then removing all containers in the pod | 

### Return type

[**PodRmReport**](PodRmReport.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **RestartPod**
> PodRestartReport RestartPod(ctx, name)
Restart a pod

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name** | **string**| the name or ID of the pod | 

### Return type

[**PodRestartReport**](PodRestartReport.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **StartPod**
> PodStartReport StartPod(ctx, name)
Start a pod

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name** | **string**| the name or ID of the pod | 

### Return type

[**PodStartReport**](PodStartReport.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **StatsPod**
> InlineResponse2002 StatsPod(ctx, optional)
Get stats for one or more pods

Display a live stream of resource usage statistics for the containers in one or more pods

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***PodsApiStatsPodOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a PodsApiStatsPodOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **all** | **optional.Bool**| Provide statistics for all running pods. | 
 **namesOrIDs** | [**optional.Interface of []string**](string.md)| Names or IDs of pods. | 

### Return type

[**InlineResponse2002**](inline_response_200_2.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **StopPod**
> PodStopReport StopPod(ctx, name, optional)
Stop a pod

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name** | **string**| the name or ID of the pod | 
 **optional** | ***PodsApiStopPodOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a PodsApiStopPodOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **t** | **optional.Int32**| timeout | 

### Return type

[**PodStopReport**](PodStopReport.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **TopPod**
> InlineResponse2002 TopPod(ctx, name, optional)
List processes

List processes running inside a pod

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name** | **string**| Name of pod to query for processes  | 
 **optional** | ***PodsApiTopPodOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a PodsApiTopPodOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **stream** | **optional.Bool**| Stream the output | [default to true]
 **psArgs** | **optional.String**| arguments to pass to ps such as aux. Requires ps(1) to be installed in the container if no ps(1) compatible AIX descriptors are used. | [default to -ef]

### Return type

[**InlineResponse2002**](inline_response_200_2.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UnpausePod**
> PodUnpauseReport UnpausePod(ctx, name)
Unpause a pod

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name** | **string**| the name or ID of the pod | 

### Return type

[**PodUnpauseReport**](PodUnpauseReport.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

