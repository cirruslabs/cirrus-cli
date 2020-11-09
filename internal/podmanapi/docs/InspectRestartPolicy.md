# InspectRestartPolicy

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**MaximumRetryCount** | **int32** | MaximumRetryCount is the maximum number of retries allowed if the \&quot;on-failure\&quot; restart policy is in use. Not used if \&quot;on-failure\&quot; is not set. | [optional] [default to null]
**Name** | **string** | Name contains the container&#x27;s restart policy. Allowable values are \&quot;no\&quot; or \&quot;\&quot; (take no action), \&quot;on-failure\&quot; (restart on non-zero exit code, with an optional max retry count), and \&quot;always\&quot; (always restart on container stop, unless explicitly requested by API). Note that this is NOT actually a name of any sort - the poor naming is for Docker compatibility. | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

