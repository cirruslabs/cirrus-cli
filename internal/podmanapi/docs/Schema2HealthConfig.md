# Schema2HealthConfig

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Interval** | [***Duration**](Duration.md) |  | [optional] [default to null]
**Retries** | **int64** | Retries is the number of consecutive failures needed to consider a container as unhealthy. Zero means inherit. | [optional] [default to null]
**StartPeriod** | [***Duration**](Duration.md) |  | [optional] [default to null]
**Test** | **[]string** | Test is the test to perform to check that the container is healthy. An empty slice means to inherit the default. The options are: {} : inherit healthcheck {\&quot;NONE\&quot;} : disable healthcheck {\&quot;CMD\&quot;, args...} : exec arguments directly {\&quot;CMD-SHELL\&quot;, command} : run command with system&#39;s default shell | [optional] [default to null]
**Timeout** | [***Duration**](Duration.md) |  | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


