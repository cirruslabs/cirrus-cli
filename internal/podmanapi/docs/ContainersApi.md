# {{classname}}

All URIs are relative to *http://podman.io/*

Method | HTTP request | Description
------------- | ------------- | -------------
[**ChangesContainer**](ContainersApi.md#ChangesContainer) | **Get** /libpod/containers/{name}/changes | Report on changes to container&#x27;s filesystem; adds, deletes or modifications.
[**LibpodAttachContainer**](ContainersApi.md#LibpodAttachContainer) | **Post** /libpod/containers/{name}/attach | Attach to a container
[**LibpodCheckpointContainer**](ContainersApi.md#LibpodCheckpointContainer) | **Post** /libpod/containers/{name}/checkpoint | Checkpoint a container
[**LibpodCommitContainer**](ContainersApi.md#LibpodCommitContainer) | **Post** /libpod/commit | Commit
[**LibpodContainerExists**](ContainersApi.md#LibpodContainerExists) | **Get** /libpod/containers/{name}/exists | Check if container exists
[**LibpodCreateContainer**](ContainersApi.md#LibpodCreateContainer) | **Post** /libpod/containers/create | Create a container
[**LibpodExportContainer**](ContainersApi.md#LibpodExportContainer) | **Get** /libpod/containers/{name}/export | Export a container
[**LibpodGenerateKube**](ContainersApi.md#LibpodGenerateKube) | **Get** /libpod/generate/{name:.*}/kube | Generate a Kubernetes YAML file.
[**LibpodGenerateSystemd**](ContainersApi.md#LibpodGenerateSystemd) | **Get** /libpod/generate/{name:.*}/systemd | Generate Systemd Units
[**LibpodGetContainer**](ContainersApi.md#LibpodGetContainer) | **Get** /libpod/containers/{name}/json | Inspect container
[**LibpodInitContainer**](ContainersApi.md#LibpodInitContainer) | **Post** /libpod/containers/{name}/init | Initialize a container
[**LibpodKillContainer**](ContainersApi.md#LibpodKillContainer) | **Post** /libpod/containers/{name}/kill | Kill container
[**LibpodListContainers**](ContainersApi.md#LibpodListContainers) | **Get** /libpod/containers/json | List containers
[**LibpodLogsFromContainer**](ContainersApi.md#LibpodLogsFromContainer) | **Get** /libpod/containers/{name}/logs | Get container logs
[**LibpodMountContainer**](ContainersApi.md#LibpodMountContainer) | **Post** /libpod/containers/{name}/mount | Mount a container
[**LibpodPauseContainer**](ContainersApi.md#LibpodPauseContainer) | **Post** /libpod/containers/{name}/pause | Pause a container
[**LibpodPlayKube**](ContainersApi.md#LibpodPlayKube) | **Post** /libpod/play/kube | Play a Kubernetes YAML file.
[**LibpodPruneContainers**](ContainersApi.md#LibpodPruneContainers) | **Post** /libpod/containers/prune | Delete stopped containers
[**LibpodPutArchive**](ContainersApi.md#LibpodPutArchive) | **Post** /libpod/containers/{name}/copy | Copy files into a container
[**LibpodRemoveContainer**](ContainersApi.md#LibpodRemoveContainer) | **Delete** /libpod/containers/{name} | Delete container
[**LibpodResizeContainer**](ContainersApi.md#LibpodResizeContainer) | **Post** /libpod/containers/{name}/resize | Resize a container&#x27;s TTY
[**LibpodRestartContainer**](ContainersApi.md#LibpodRestartContainer) | **Post** /libpod/containers/{name}/restart | Restart a container
[**LibpodRestoreContainer**](ContainersApi.md#LibpodRestoreContainer) | **Post** /libpod/containers/{name}/restore | Restore a container
[**LibpodRunHealthCheck**](ContainersApi.md#LibpodRunHealthCheck) | **Get** /libpod/containers/{name:.*}/healthcheck | Run a container&#x27;s healthcheck
[**LibpodShowMountedContainers**](ContainersApi.md#LibpodShowMountedContainers) | **Get** /libpod/containers/showmounted | Show mounted containers
[**LibpodStartContainer**](ContainersApi.md#LibpodStartContainer) | **Post** /libpod/containers/{name}/start | Start a container
[**LibpodStatsContainer**](ContainersApi.md#LibpodStatsContainer) | **Get** /libpod/containers/{name}/stats | Get stats for a container
[**LibpodStatsContainers**](ContainersApi.md#LibpodStatsContainers) | **Get** /libpod/containers/stats | Get stats for one or more containers
[**LibpodStopContainer**](ContainersApi.md#LibpodStopContainer) | **Post** /libpod/containers/{name}/stop | Stop a container
[**LibpodTopContainer**](ContainersApi.md#LibpodTopContainer) | **Get** /libpod/containers/{name}/top | List processes
[**LibpodUnmountContainer**](ContainersApi.md#LibpodUnmountContainer) | **Post** /libpod/containers/{name}/unmount | Unmount a container
[**LibpodUnpauseContainer**](ContainersApi.md#LibpodUnpauseContainer) | **Post** /libpod/containers/{name}/unpause | Unpause Container
[**LibpodWaitContainer**](ContainersApi.md#LibpodWaitContainer) | **Post** /libpod/containers/{name}/wait | Wait on a container

# **ChangesContainer**
> ChangesContainer(ctx, name)
Report on changes to container's filesystem; adds, deletes or modifications.

Returns which files in a container's filesystem have been added, deleted, or modified. The Kind of modification can be one of:  0: Modified 1: Added 2: Deleted 

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name** | **string**| the name or id of the container | 

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, text/plain, text/html

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **LibpodAttachContainer**
> LibpodAttachContainer(ctx, name, optional)
Attach to a container

Hijacks the connection to forward the container's standard streams to the client.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name** | **string**| the name or ID of the container | 
 **optional** | ***ContainersApiLibpodAttachContainerOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a ContainersApiLibpodAttachContainerOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **detachKeys** | **optional.String**| keys to use for detaching from the container | 
 **logs** | **optional.Bool**| Stream all logs from the container across the connection. Happens before streaming attach (if requested). At least one of logs or stream must be set | 
 **stream** | **optional.Bool**| Attach to the container. If unset, and logs is set, only the container&#x27;s logs will be sent. At least one of stream or logs must be set | [default to true]
 **stdout** | **optional.Bool**| Attach to container STDOUT | 
 **stderr** | **optional.Bool**| Attach to container STDERR | 
 **stdin** | **optional.Bool**| Attach to container STDIN | 

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **LibpodCheckpointContainer**
> LibpodCheckpointContainer(ctx, name, optional)
Checkpoint a container

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name** | **string**| the name or ID of the container | 
 **optional** | ***ContainersApiLibpodCheckpointContainerOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a ContainersApiLibpodCheckpointContainerOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **keep** | **optional.Bool**| keep all temporary checkpoint files | 
 **leaveRunning** | **optional.Bool**| leave the container running after writing checkpoint to disk | 
 **tcpEstablished** | **optional.Bool**| checkpoint a container with established TCP connections | 
 **export** | **optional.Bool**| export the checkpoint image to a tar.gz | 
 **ignoreRootFS** | **optional.Bool**| do not include root file-system changes when exporting | 

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **LibpodCommitContainer**
> LibpodCommitContainer(ctx, container, optional)
Commit

Create a new image from a container

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **container** | **string**| the name or ID of a container | 
 **optional** | ***ContainersApiLibpodCommitContainerOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a ContainersApiLibpodCommitContainerOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **repo** | **optional.String**| the repository name for the created image | 
 **tag** | **optional.String**| tag name for the created image | 
 **comment** | **optional.String**| commit message | 
 **author** | **optional.String**| author of the image | 
 **pause** | **optional.Bool**| pause the container before committing it | 
 **changes** | [**optional.Interface of []string**](string.md)| instructions to apply while committing in Dockerfile format (i.e. \&quot;CMD&#x3D;/bin/foo\&quot;) | 
 **format** | **optional.String**| format of the image manifest and metadata (default \&quot;oci\&quot;) | 

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **LibpodContainerExists**
> LibpodContainerExists(ctx, name)
Check if container exists

Quick way to determine if a container exists by name or ID

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name** | **string**| the name or ID of the container | 

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **LibpodCreateContainer**
> InlineResponse201 LibpodCreateContainer(ctx, optional)
Create a container

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***ContainersApiLibpodCreateContainerOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a ContainersApiLibpodCreateContainerOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**optional.Interface of SpecGenerator**](SpecGenerator.md)| attributes for creating a container | 

### Return type

[**InlineResponse201**](inline_response_201.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json, application/x-tar
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **LibpodExportContainer**
> LibpodExportContainer(ctx, name)
Export a container

Export the contents of a container as a tarball.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name** | **string**| the name or ID of the container | 

### Return type

 (empty response body)

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
 **optional** | ***ContainersApiLibpodGenerateKubeOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a ContainersApiLibpodGenerateKubeOpts struct
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
 **optional** | ***ContainersApiLibpodGenerateSystemdOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a ContainersApiLibpodGenerateSystemdOpts struct
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

# **LibpodGetContainer**
> InlineResponse2008 LibpodGetContainer(ctx, name, optional)
Inspect container

Return low-level information about a container.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name** | **string**| the name or ID of the container | 
 **optional** | ***ContainersApiLibpodGetContainerOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a ContainersApiLibpodGetContainerOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **size** | **optional.Bool**| display filesystem usage | 

### Return type

[**InlineResponse2008**](inline_response_200_8.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **LibpodInitContainer**
> LibpodInitContainer(ctx, name)
Initialize a container

Performs all tasks necessary for initializing the container but does not start the container.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name** | **string**| the name or ID of the container | 

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **LibpodKillContainer**
> LibpodKillContainer(ctx, name, optional)
Kill container

send a signal to a container, defaults to killing the container

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name** | **string**| the name or ID of the container | 
 **optional** | ***ContainersApiLibpodKillContainerOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a ContainersApiLibpodKillContainerOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **signal** | **optional.String**| signal to be sent to container, either by integer or SIG_ name | [default to TERM]

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **LibpodListContainers**
> []ListContainer LibpodListContainers(ctx, optional)
List containers

Returns a list of containers

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***ContainersApiLibpodListContainersOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a ContainersApiLibpodListContainersOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **all** | **optional.Bool**| Return all containers. By default, only running containers are shown | [default to false]
 **limit** | **optional.Int32**| Return this number of most recently created containers, including non-running ones. | 
 **pod** | **optional.Bool**| Ignored. Previously included details on pod name and ID that are currently included by default. | [default to false]
 **size** | **optional.Bool**| Return the size of container as fields SizeRw and SizeRootFs. | [default to false]
 **sync** | **optional.Bool**| Sync container state with OCI runtime | [default to false]
 **filters** | **optional.String**| A JSON encoded value of the filters (a &#x60;map[string][]string&#x60;) to process on the containers list. Available filters: - &#x60;ancestor&#x60;&#x3D;(&#x60;&lt;image-name&gt;[:&lt;tag&gt;]&#x60;, &#x60;&lt;image id&gt;&#x60;, or &#x60;&lt;image@digest&gt;&#x60;) - &#x60;before&#x60;&#x3D;(&#x60;&lt;container id&gt;&#x60; or &#x60;&lt;container name&gt;&#x60;) - &#x60;expose&#x60;&#x3D;(&#x60;&lt;port&gt;[/&lt;proto&gt;]&#x60; or &#x60;&lt;startport-endport&gt;/[&lt;proto&gt;]&#x60;) - &#x60;exited&#x3D;&lt;int&gt;&#x60; containers with exit code of &#x60;&lt;int&gt;&#x60; - &#x60;health&#x60;&#x3D;(&#x60;starting&#x60;, &#x60;healthy&#x60;, &#x60;unhealthy&#x60; or &#x60;none&#x60;) - &#x60;id&#x3D;&lt;ID&gt;&#x60; a container&#x27;s ID - &#x60;is-task&#x60;&#x3D;(&#x60;true&#x60; or &#x60;false&#x60;) - &#x60;label&#x60;&#x3D;(&#x60;key&#x60; or &#x60;\&quot;key&#x3D;value\&quot;&#x60;) of an container label - &#x60;name&#x3D;&lt;name&gt;&#x60; a container&#x27;s name - &#x60;network&#x60;&#x3D;(&#x60;&lt;network id&gt;&#x60; or &#x60;&lt;network name&gt;&#x60;) - &#x60;publish&#x60;&#x3D;(&#x60;&lt;port&gt;[/&lt;proto&gt;]&#x60; or &#x60;&lt;startport-endport&gt;/[&lt;proto&gt;]&#x60;) - &#x60;since&#x60;&#x3D;(&#x60;&lt;container id&gt;&#x60; or &#x60;&lt;container name&gt;&#x60;) - &#x60;status&#x60;&#x3D;(&#x60;created&#x60;, &#x60;restarting&#x60;, &#x60;running&#x60;, &#x60;removing&#x60;, &#x60;paused&#x60;, &#x60;exited&#x60; or &#x60;dead&#x60;) - &#x60;volume&#x60;&#x3D;(&#x60;&lt;volume name&gt;&#x60; or &#x60;&lt;mount point destination&gt;&#x60;)  | 

### Return type

[**[]ListContainer**](ListContainer.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **LibpodLogsFromContainer**
> LibpodLogsFromContainer(ctx, name, optional)
Get container logs

Get stdout and stderr logs from a container.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name** | **string**| the name or ID of the container | 
 **optional** | ***ContainersApiLibpodLogsFromContainerOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a ContainersApiLibpodLogsFromContainerOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **follow** | **optional.Bool**| Keep connection after returning logs. | 
 **stdout** | **optional.Bool**| Return logs from stdout | 
 **stderr** | **optional.Bool**| Return logs from stderr | 
 **since** | **optional.String**| Only return logs since this time, as a UNIX timestamp | 
 **until** | **optional.String**| Only return logs before this time, as a UNIX timestamp | 
 **timestamps** | **optional.Bool**| Add timestamps to every log line | [default to false]
 **tail** | **optional.String**| Only return this number of log lines from the end of the logs | [default to all]

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **LibpodMountContainer**
> string LibpodMountContainer(ctx, name)
Mount a container

Mount a container to the filesystem

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name** | **string**| the name or ID of the container | 

### Return type

**string**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **LibpodPauseContainer**
> LibpodPauseContainer(ctx, name)
Pause a container

Use the cgroups freezer to suspend all processes in a container.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name** | **string**| the name or ID of the container | 

### Return type

 (empty response body)

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
 **optional** | ***ContainersApiLibpodPlayKubeOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a ContainersApiLibpodPlayKubeOpts struct
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

# **LibpodPruneContainers**
> []LibpodContainersPruneReport LibpodPruneContainers(ctx, optional)
Delete stopped containers

Remove containers not in use

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***ContainersApiLibpodPruneContainersOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a ContainersApiLibpodPruneContainersOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **filters** | **optional.String**| Filters to process on the prune list, encoded as JSON (a &#x60;map[string][]string&#x60;).  Available filters:  - &#x60;until&#x3D;&lt;timestamp&gt;&#x60; Prune containers created before this timestamp. The &#x60;&lt;timestamp&gt;&#x60; can be Unix timestamps, date formatted timestamps, or Go duration strings (e.g. &#x60;10m&#x60;, &#x60;1h30m&#x60;) computed relative to the daemon machine’s time.  - &#x60;label&#x60; (&#x60;label&#x3D;&lt;key&gt;&#x60;, &#x60;label&#x3D;&lt;key&gt;&#x3D;&lt;value&gt;&#x60;, &#x60;label!&#x3D;&lt;key&gt;&#x60;, or &#x60;label!&#x3D;&lt;key&gt;&#x3D;&lt;value&gt;&#x60;) Prune containers with (or without, in case &#x60;label!&#x3D;...&#x60; is used) the specified labels.  | 

### Return type

[**[]LibpodContainersPruneReport**](LibpodContainersPruneReport.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **LibpodPutArchive**
> LibpodPutArchive(ctx, name, path, optional)
Copy files into a container

Copy a tar archive of files into a container

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name** | **string**| container name or id | 
  **path** | **string**| Path to a directory in the container to extract | 
 **optional** | ***ContainersApiLibpodPutArchiveOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a ContainersApiLibpodPutArchiveOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


 **body** | [**optional.Interface of string**](string.md)| tarfile of files to copy into the container | 
 **pause** | **optional.**| pause the container while copying (defaults to true) | [default to true]

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json, application/x-tar
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **LibpodRemoveContainer**
> LibpodRemoveContainer(ctx, name, optional)
Delete container

Delete container

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name** | **string**| the name or ID of the container | 
 **optional** | ***ContainersApiLibpodRemoveContainerOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a ContainersApiLibpodRemoveContainerOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **force** | **optional.Bool**| need something | 
 **v** | **optional.Bool**| delete volumes | 

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **LibpodResizeContainer**
> interface{} LibpodResizeContainer(ctx, name, optional)
Resize a container's TTY

Resize the terminal attached to a container (for use with Attach).

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name** | **string**| the name or ID of the container | 
 **optional** | ***ContainersApiLibpodResizeContainerOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a ContainersApiLibpodResizeContainerOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **h** | **optional.Int32**| Height to set for the terminal, in characters | 
 **w** | **optional.Int32**| Width to set for the terminal, in characters | 

### Return type

[**interface{}**](interface{}.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **LibpodRestartContainer**
> LibpodRestartContainer(ctx, name, optional)
Restart a container

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name** | **string**| the name or ID of the container | 
 **optional** | ***ContainersApiLibpodRestartContainerOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a ContainersApiLibpodRestartContainerOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **t** | **optional.Int32**| timeout before sending kill signal to container | 

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **LibpodRestoreContainer**
> LibpodRestoreContainer(ctx, name, optional)
Restore a container

Restore a container from a checkpoint.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name** | **string**| the name or id of the container | 
 **optional** | ***ContainersApiLibpodRestoreContainerOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a ContainersApiLibpodRestoreContainerOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **name** | **optional.String**| the name of the container when restored from a tar. can only be used with import | 
 **keep** | **optional.Bool**| keep all temporary checkpoint files | 
 **leaveRunning** | **optional.Bool**| leave the container running after writing checkpoint to disk | 
 **tcpEstablished** | **optional.Bool**| checkpoint a container with established TCP connections | 
 **import_** | **optional.Bool**| import the restore from a checkpoint tar.gz | 
 **ignoreRootFS** | **optional.Bool**| do not include root file-system changes when exporting | 
 **ignoreStaticIP** | **optional.Bool**| ignore IP address if set statically | 
 **ignoreStaticMAC** | **optional.Bool**| ignore MAC address if set statically | 

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **LibpodRunHealthCheck**
> InlineResponse2007 LibpodRunHealthCheck(ctx, name_)
Run a container's healthcheck

Execute the defined healthcheck and return information about the results

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name_** | **string**| the name or ID of the container | 

### Return type

[**InlineResponse2007**](inline_response_200_7.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **LibpodShowMountedContainers**
> map[string]string LibpodShowMountedContainers(ctx, )
Show mounted containers

Lists all mounted containers mount points

### Required Parameters
This endpoint does not need any parameter.

### Return type

**map[string]string**

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **LibpodStartContainer**
> LibpodStartContainer(ctx, name, optional)
Start a container

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name** | **string**| the name or ID of the container | 
 **optional** | ***ContainersApiLibpodStartContainerOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a ContainersApiLibpodStartContainerOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **detachKeys** | **optional.String**| Override the key sequence for detaching a container. Format is a single character [a-Z] or ctrl-&lt;value&gt; where &lt;value&gt; is one of: a-z, @, ^, [, , or _. | [default to ctrl-p,ctrl-q]

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **LibpodStatsContainer**
> LibpodStatsContainer(ctx, name, optional)
Get stats for a container

DEPRECATED. This endpoint will be removed with the next major release. Please use /libpod/containers/stats instead.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name** | **string**| the name or ID of the container | 
 **optional** | ***ContainersApiLibpodStatsContainerOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a ContainersApiLibpodStatsContainerOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **stream** | **optional.Bool**| Stream the output | [default to true]

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **LibpodStatsContainers**
> LibpodStatsContainers(ctx, optional)
Get stats for one or more containers

Return a live stream of resource usage statistics of one or more container. If no container is specified, the statistics of all containers are returned.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***ContainersApiLibpodStatsContainersOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a ContainersApiLibpodStatsContainersOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **containers** | [**optional.Interface of []string**](string.md)| names or IDs of containers | 
 **stream** | **optional.Bool**| Stream the output | [default to true]

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **LibpodStopContainer**
> LibpodStopContainer(ctx, name, optional)
Stop a container

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name** | **string**| the name or ID of the container | 
 **optional** | ***ContainersApiLibpodStopContainerOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a ContainersApiLibpodStopContainerOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **t** | **optional.Int32**| number of seconds to wait before killing container | 

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **LibpodTopContainer**
> InlineResponse2002 LibpodTopContainer(ctx, name, optional)
List processes

List processes running inside a container

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name** | **string**| Name of container to query for processes (As of version 1.xx)  | 
 **optional** | ***ContainersApiLibpodTopContainerOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a ContainersApiLibpodTopContainerOpts struct
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

# **LibpodUnmountContainer**
> LibpodUnmountContainer(ctx, name)
Unmount a container

Unmount a container from the filesystem

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name** | **string**| the name or ID of the container | 

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **LibpodUnpauseContainer**
> LibpodUnpauseContainer(ctx, name)
Unpause Container

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name** | **string**| the name or ID of the container | 

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **LibpodWaitContainer**
> InlineResponse2003 LibpodWaitContainer(ctx, name, optional)
Wait on a container

Wait on a container to met a given condition

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name** | **string**| the name or ID of the container | 
 **optional** | ***ContainersApiLibpodWaitContainerOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a ContainersApiLibpodWaitContainerOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **condition** | **optional.String**| wait until container is to a given condition. default is stopped. valid conditions are:   - configured   - created   - exited   - paused   - running   - stopped  | 

### Return type

[**InlineResponse2003**](inline_response_200_3.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

