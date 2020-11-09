# ImageConfig

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Cmd** | **[]string** | Cmd defines the default arguments to the entrypoint of the container. | [optional] [default to null]
**Entrypoint** | **[]string** | Entrypoint defines a list of arguments to use as the command to execute when the container starts. | [optional] [default to null]
**Env** | **[]string** | Env is a list of environment variables to be used in a container. | [optional] [default to null]
**ExposedPorts** | [**map[string]interface{}**](interface{}.md) | ExposedPorts a set of ports to expose from a container running this image. | [optional] [default to null]
**Labels** | **map[string]string** | Labels contains arbitrary metadata for the container. | [optional] [default to null]
**StopSignal** | **string** | StopSignal contains the system call signal that will be sent to the container to exit. | [optional] [default to null]
**User** | **string** | User defines the username or UID which the process in the container should run as. | [optional] [default to null]
**Volumes** | [**map[string]interface{}**](interface{}.md) | Volumes is a set of directories describing where the process is likely write data specific to a container instance. | [optional] [default to null]
**WorkingDir** | **string** | WorkingDir sets the current working directory of the entrypoint process in the container. | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

