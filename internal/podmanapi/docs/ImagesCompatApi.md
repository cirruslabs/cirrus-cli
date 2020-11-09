# {{classname}}

All URIs are relative to *http://podman.io/*

Method | HTTP request | Description
------------- | ------------- | -------------
[**BuildImage**](ImagesCompatApi.md#BuildImage) | **Post** /build | Create image
[**CreateImage**](ImagesCompatApi.md#CreateImage) | **Post** /images/create | Create an image
[**ExportImage**](ImagesCompatApi.md#ExportImage) | **Get** /images/{name:.*}/get | Export an image
[**ImageHistory**](ImagesCompatApi.md#ImageHistory) | **Get** /images/{name:.*}/history | History of an image
[**ImportImage**](ImagesCompatApi.md#ImportImage) | **Post** /images/load | Import image
[**InspectImage**](ImagesCompatApi.md#InspectImage) | **Get** /images/{name:.*}/json | Inspect an image
[**ListImages**](ImagesCompatApi.md#ListImages) | **Get** /images/json | List Images
[**PruneImages**](ImagesCompatApi.md#PruneImages) | **Post** /images/prune | Prune unused images
[**PushImage**](ImagesCompatApi.md#PushImage) | **Post** /images/{name:.*}/push | Push Image
[**RemoveImage**](ImagesCompatApi.md#RemoveImage) | **Delete** /images/{name:.*} | Remove Image
[**SearchImages**](ImagesCompatApi.md#SearchImages) | **Get** /images/search | Search images
[**TagImage**](ImagesCompatApi.md#TagImage) | **Post** /images/{name:.*}/tag | Tag an image

# **BuildImage**
> InlineResponse200 BuildImage(ctx, optional)
Create image

Build an image from the given Dockerfile(s)

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***ImagesCompatApiBuildImageOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a ImagesCompatApiBuildImageOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **dockerfile** | **optional.String**| Path within the build context to the &#x60;Dockerfile&#x60;. This is ignored if remote is specified and points to an external &#x60;Dockerfile&#x60;.  | [default to Dockerfile]
 **t** | **optional.String**| A name and optional tag to apply to the image in the &#x60;name:tag&#x60; format. If you omit the tag the default latest value is assumed. You can provide several t parameters. | [default to latest]
 **extrahosts** | **optional.String**| TBD Extra hosts to add to /etc/hosts (As of version 1.xx)  | 
 **remote** | **optional.String**| A Git repository URI or HTTP/HTTPS context URI. If the URI points to a single text file, the file’s contents are placed into a file called Dockerfile and the image is built from that file. If the URI points to a tarball, the file is downloaded by the daemon and the contents therein used as the context for the build. If the URI points to a tarball and the dockerfile parameter is also specified, there must be a file with the corresponding path inside the tarball. (As of version 1.xx)  | 
 **q** | **optional.Bool**| Suppress verbose build output  | [default to false]
 **nocache** | **optional.Bool**| Do not use the cache when building the image (As of version 1.xx)  | [default to false]
 **cachefrom** | **optional.String**| JSON array of images used to build cache resolution (As of version 1.xx)  | 
 **pull** | **optional.Bool**| Attempt to pull the image even if an older image exists locally (As of version 1.xx)  | [default to false]
 **rm** | **optional.Bool**| Remove intermediate containers after a successful build (As of version 1.xx)  | [default to true]
 **forcerm** | **optional.Bool**| Always remove intermediate containers, even upon failure (As of version 1.xx)  | [default to false]
 **memory** | **optional.Int32**| Memory is the upper limit (in bytes) on how much memory running containers can use (As of version 1.xx)  | 
 **memswap** | **optional.Int32**| MemorySwap limits the amount of memory and swap together (As of version 1.xx)  | 
 **cpushares** | **optional.Int32**| CPUShares (relative weight (As of version 1.xx)  | 
 **cpusetcpus** | **optional.String**| CPUSetCPUs in which to allow execution (0-3, 0,1) (As of version 1.xx)  | 
 **cpuperiod** | **optional.Int32**| CPUPeriod limits the CPU CFS (Completely Fair Scheduler) period (As of version 1.xx)  | 
 **cpuquota** | **optional.Int32**| CPUQuota limits the CPU CFS (Completely Fair Scheduler) quota (As of version 1.xx)  | 
 **buildargs** | **optional.String**| JSON map of string pairs denoting build-time variables. For example, the build argument &#x60;Foo&#x60; with the value of &#x60;bar&#x60; would be encoded in JSON as &#x60;[\&quot;Foo\&quot;:\&quot;bar\&quot;]&#x60;.  For example, buildargs&#x3D;{\&quot;Foo\&quot;:\&quot;bar\&quot;}.  Note(s): * This should not be used to pass secrets. * The value of buildargs should be URI component encoded before being passed to the API.  (As of version 1.xx)  | 
 **shmsize** | **optional.Int32**| ShmSize is the \&quot;size\&quot; value to use when mounting an shmfs on the container&#x27;s /dev/shm directory. Default is 64MB (As of version 1.xx)  | [default to 67108864]
 **squash** | **optional.Bool**| Silently ignored. Squash the resulting images layers into a single layer (As of version 1.xx)  | [default to false]
 **labels** | **optional.String**| JSON map of key, value pairs to set as labels on the new image (As of version 1.xx)  | 
 **networkmode** | **optional.String**| Sets the networking mode for the run commands during build. Supported standard values are:   * &#x60;bridge&#x60; limited to containers within a single host, port mapping required for external access   * &#x60;host&#x60; no isolation between host and containers on this network   * &#x60;none&#x60; disable all networking for this container   * container:&lt;nameOrID&gt; share networking with given container   ---All other values are assumed to be a custom network&#x27;s name (As of version 1.xx)  | [default to bridge]
 **platform** | **optional.String**| Platform format os[/arch[/variant]] (As of version 1.xx)  | 
 **target** | **optional.String**| Target build stage (As of version 1.xx)  | 
 **outputs** | **optional.String**| output configuration TBD (As of version 1.xx)  | 

### Return type

[**InlineResponse200**](inline_response_200.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **CreateImage**
> interface{} CreateImage(ctx, optional)
Create an image

Create an image by either pulling it from a registry or importing it.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***ImagesCompatApiCreateImageOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a ImagesCompatApiCreateImageOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**optional.Interface of string**](string.md)| Image content if fromSrc parameter was used | 
 **fromImage** | **optional.**| needs description | 
 **fromSrc** | **optional.**| needs description | 
 **tag** | **optional.**| needs description | 
 **xRegistryAuth** | **optional.**| A base64-encoded auth configuration. | 

### Return type

[**interface{}**](interface{}.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json, application/x-tar
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ExportImage**
> *os.File ExportImage(ctx, name_)
Export an image

Export an image in tarball format

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name_** | **string**| the name or ID of the container | 

### Return type

[***os.File**](*os.File.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ImageHistory**
> InlineResponse2004 ImageHistory(ctx, name_)
History of an image

Return parent layers of an image.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name_** | **string**| the name or ID of the container | 

### Return type

[**InlineResponse2004**](inline_response_200_4.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ImportImage**
> ImportImage(ctx, optional)
Import image

Load a set of images and tags into a repository.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***ImagesCompatApiImportImageOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a ImagesCompatApiImportImageOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**optional.Interface of string**](string.md)| tarball of container image | 
 **quiet** | **optional.**| not supported | 

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json, application/x-tar
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **InspectImage**
> InlineResponse2005 InspectImage(ctx, name_)
Inspect an image

Return low-level information about an image.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name_** | **string**| the name or ID of the container | 

### Return type

[**InlineResponse2005**](inline_response_200_5.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ListImages**
> []ImageSummary ListImages(ctx, optional)
List Images

Returns a list of images on the server. Note that it uses a different, smaller representation of an image than inspecting a single image.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***ImagesCompatApiListImagesOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a ImagesCompatApiListImagesOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **all** | **optional.Bool**| Show all images. Only images from a final layer (no children) are shown by default. | [default to false]
 **filters** | **optional.String**| A JSON encoded value of the filters (a &#x60;map[string][]string&#x60;) to process on the images list. Available filters: - &#x60;before&#x60;&#x3D;(&#x60;&lt;image-name&gt;[:&lt;tag&gt;]&#x60;,  &#x60;&lt;image id&gt;&#x60; or &#x60;&lt;image@digest&gt;&#x60;) - &#x60;dangling&#x3D;true&#x60; - &#x60;label&#x3D;key&#x60; or &#x60;label&#x3D;\&quot;key&#x3D;value\&quot;&#x60; of an image label - &#x60;reference&#x60;&#x3D;(&#x60;&lt;image-name&gt;[:&lt;tag&gt;]&#x60;) - &#x60;since&#x60;&#x3D;(&#x60;&lt;image-name&gt;[:&lt;tag&gt;]&#x60;,  &#x60;&lt;image id&gt;&#x60; or &#x60;&lt;image@digest&gt;&#x60;)  | 
 **digests** | **optional.Bool**| Not supported | [default to false]

### Return type

[**[]ImageSummary**](ImageSummary.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **PruneImages**
> []ImageDeleteResponse PruneImages(ctx, optional)
Prune unused images

Remove images from local storage that are not being used by a container

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***ImagesCompatApiPruneImagesOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a ImagesCompatApiPruneImagesOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **filters** | **optional.String**| filters to apply to image pruning, encoded as JSON (map[string][]string). Available filters:   - &#x60;dangling&#x3D;&lt;boolean&gt;&#x60; When set to &#x60;true&#x60; (or &#x60;1&#x60;), prune only      unused *and* untagged images. When set to &#x60;false&#x60;      (or &#x60;0&#x60;), all unused images are pruned.   - &#x60;until&#x3D;&lt;string&gt;&#x60; Prune images created before this timestamp. The &#x60;&lt;timestamp&gt;&#x60; can be Unix timestamps, date formatted timestamps, or Go duration strings (e.g. &#x60;10m&#x60;, &#x60;1h30m&#x60;) computed relative to the daemon machine’s time.   - &#x60;label&#x60; (&#x60;label&#x3D;&lt;key&gt;&#x60;, &#x60;label&#x3D;&lt;key&gt;&#x3D;&lt;value&gt;&#x60;, &#x60;label!&#x3D;&lt;key&gt;&#x60;, or &#x60;label!&#x3D;&lt;key&gt;&#x3D;&lt;value&gt;&#x60;) Prune images with (or without, in case &#x60;label!&#x3D;...&#x60; is used) the specified labels.  | 

### Return type

[**[]ImageDeleteResponse**](ImageDeleteResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **PushImage**
> *os.File PushImage(ctx, name_, optional)
Push Image

Push an image to a container registry

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name_** | **string**| Name of image to push. | 
 **optional** | ***ImagesCompatApiPushImageOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a ImagesCompatApiPushImageOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **tag** | **optional.String**| The tag to associate with the image on the registry. | 
 **xRegistryAuth** | **optional.String**| A base64-encoded auth configuration. | 

### Return type

[***os.File**](*os.File.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **RemoveImage**
> []ImageDeleteResponse RemoveImage(ctx, name_, optional)
Remove Image

Delete an image from local storage

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name_** | **string**| name or ID of image to delete | 
 **optional** | ***ImagesCompatApiRemoveImageOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a ImagesCompatApiRemoveImageOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **force** | **optional.Bool**| remove the image even if used by containers or has other tags | 
 **noprune** | **optional.Bool**| not supported. will be logged as an invalid parameter if enabled | 

### Return type

[**[]ImageDeleteResponse**](ImageDeleteResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **SearchImages**
> InlineResponse2006 SearchImages(ctx, optional)
Search images

Search registries for an image

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***ImagesCompatApiSearchImagesOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a ImagesCompatApiSearchImagesOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **term** | **optional.String**| term to search | 
 **limit** | **optional.Int32**| maximum number of results | 
 **filters** | **optional.String**| A JSON encoded value of the filters (a &#x60;map[string][]string&#x60;) to process on the images list. Available filters: - &#x60;is-automated&#x3D;(true|false)&#x60; - &#x60;is-official&#x3D;(true|false)&#x60; - &#x60;stars&#x3D;&lt;number&gt;&#x60; Matches images that has at least &#x27;number&#x27; stars.  | 

### Return type

[**InlineResponse2006**](inline_response_200_6.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **TagImage**
> TagImage(ctx, name_, optional)
Tag an image

Tag an image so that it becomes part of a repository.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **name_** | **string**| the name or ID of the container | 
 **optional** | ***ImagesCompatApiTagImageOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a ImagesCompatApiTagImageOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **repo** | **optional.String**| the repository to tag in | 
 **tag** | **optional.String**| the name of the new tag | 

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

