# LinuxCpu

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Cpus** | **string** | CPUs to use within the cpuset. Default is to use any CPU available. | [optional] [default to null]
**Mems** | **string** | List of memory nodes in the cpuset. Default is to use any available memory node. | [optional] [default to null]
**Period** | **int32** | CPU period to be used for hardcapping (in usecs). | [optional] [default to null]
**Quota** | **int64** | CPU hardcap limit (in usecs). Allowed cpu time in a given period. | [optional] [default to null]
**RealtimePeriod** | **int32** | CPU period to be used for realtime scheduling (in usecs). | [optional] [default to null]
**RealtimeRuntime** | **int64** | How much time realtime scheduling may use (in usecs). | [optional] [default to null]
**Shares** | **int32** | CPU shares (relative weight (ratio) vs. other cgroups with cpu shares). | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

