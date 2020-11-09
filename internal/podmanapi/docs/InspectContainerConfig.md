# InspectContainerConfig

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Annotations** | **map[string]string** | Container annotations | [optional] [default to null]
**AttachStderr** | **bool** | Unused, at present | [optional] [default to null]
**AttachStdin** | **bool** | Unused, at present | [optional] [default to null]
**AttachStdout** | **bool** | Unused, at present | [optional] [default to null]
**Cmd** | **[]string** | Container command | [optional] [default to null]
**CreateCommand** | **[]string** | CreateCommand is the full command plus arguments of the process the container has been created with. | [optional] [default to null]
**Domainname** | **string** | Container domain name - unused at present | [optional] [default to null]
**Entrypoint** | **string** | Container entrypoint | [optional] [default to null]
**Env** | **[]string** | Container environment variables | [optional] [default to null]
**Healthcheck** | [***Schema2HealthConfig**](Schema2HealthConfig.md) |  | [optional] [default to null]
**Hostname** | **string** | Container hostname | [optional] [default to null]
**Image** | **string** | Container image | [optional] [default to null]
**Labels** | **map[string]string** | Container labels | [optional] [default to null]
**OnBuild** | **string** | On-build arguments - presently unused. More of Buildah&#x27;s domain. | [optional] [default to null]
**OpenStdin** | **bool** | Whether the container leaves STDIN open | [optional] [default to null]
**StdinOnce** | **bool** | Whether STDIN is only left open once. Presently not supported by Podman, unused. | [optional] [default to null]
**StopSignal** | **int32** | Container stop signal | [optional] [default to null]
**SystemdMode** | **bool** | SystemdMode is whether the container is running in systemd mode. In systemd mode, the container configuration is customized to optimize running systemd in the container. | [optional] [default to null]
**Timezone** | **string** | Timezone is the timezone inside the container. Local means it has the same timezone as the host machine | [optional] [default to null]
**Tty** | **bool** | Whether the container creates a TTY | [optional] [default to null]
**Umask** | **string** | Umask is the umask inside the container. | [optional] [default to null]
**User** | **string** | User the container was launched with | [optional] [default to null]
**Volumes** | [**map[string]interface{}**](interface{}.md) | Unused, at present. I&#x27;ve never seen this field populated. | [optional] [default to null]
**WorkingDir** | **string** | Container working directory | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

