# ContainerBasicConfig

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Annotations** | **map[string]string** | Annotations are key-value options passed into the container runtime that can be used to trigger special behavior. Optional. | [optional] [default to null]
**Command** | **[]string** | Command is the container&#x27;s command. If not given and Image is specified, this will be populated by the image&#x27;s configuration. Optional. | [optional] [default to null]
**ConmonPidFile** | **string** | ConmonPidFile is a path at which a PID file for Conmon will be placed. If not given, a default location will be used. Optional. | [optional] [default to null]
**ContainerCreateCommand** | **[]string** | ContainerCreateCommand is the command that was used to create this container. This will be shown in the output of Inspect() on the container, and may also be used by some tools that wish to recreate the container (e.g. &#x60;podman generate systemd --new&#x60;). Optional. | [optional] [default to null]
**Entrypoint** | **[]string** | Entrypoint is the container&#x27;s entrypoint. If not given and Image is specified, this will be populated by the image&#x27;s configuration. Optional. | [optional] [default to null]
**Env** | **map[string]string** | Env is a set of environment variables that will be set in the container. Optional. | [optional] [default to null]
**EnvHost** | **bool** | EnvHost indicates that the host environment should be added to container Optional. | [optional] [default to null]
**Hostname** | **string** | Hostname is the container&#x27;s hostname. If not set, the hostname will not be modified (if UtsNS is not private) or will be set to the container ID (if UtsNS is private). Conflicts with UtsNS if UtsNS is not set to private. Optional. | [optional] [default to null]
**Httpproxy** | **bool** | EnvHTTPProxy indicates that the http host proxy environment variables should be added to container Optional. | [optional] [default to null]
**Labels** | **map[string]string** | Labels are key-value pairs that are used to add metadata to containers. Optional. | [optional] [default to null]
**LogConfiguration** | [***LogConfig**](LogConfig.md) |  | [optional] [default to null]
**Name** | **string** | Name is the name the container will be given. If no name is provided, one will be randomly generated. Optional. | [optional] [default to null]
**Namespace** | **string** | Namespace is the libpod namespace the container will be placed in. Optional. | [optional] [default to null]
**OciRuntime** | **string** | OCIRuntime is the name of the OCI runtime that will be used to create the container. If not specified, the default will be used. Optional. | [optional] [default to null]
**Pidns** | [***Namespace**](Namespace.md) |  | [optional] [default to null]
**Pod** | **string** | Pod is the ID of the pod the container will join. Optional. | [optional] [default to null]
**RawImageName** | **string** | RawImageName is the user-specified and unprocessed input referring to a local or a remote image. | [optional] [default to null]
**Remove** | **bool** | Remove indicates if the container should be removed once it has been started and exits | [optional] [default to null]
**RestartPolicy** | **string** | RestartPolicy is the container&#x27;s restart policy - an action which will be taken when the container exits. If not given, the default policy, which does nothing, will be used. Optional. | [optional] [default to null]
**RestartTries** | **int32** | RestartRetries is the number of attempts that will be made to restart the container. Only available when RestartPolicy is set to \&quot;on-failure\&quot;. Optional. | [optional] [default to null]
**SdnotifyMode** | **string** | Determine how to handle the NOTIFY_SOCKET - do we participate or pass it through \&quot;container\&quot; - let the OCI runtime deal with it, advertise conmon&#x27;s MAINPID \&quot;conmon-only\&quot; - advertise conmon&#x27;s MAINPID, send READY when started, don&#x27;t pass to OCI \&quot;ignore\&quot; - unset NOTIFY_SOCKET | [optional] [default to null]
**Stdin** | **bool** | Stdin is whether the container will keep its STDIN open. | [optional] [default to null]
**StopSignal** | **int64** |  | [optional] [default to null]
**StopTimeout** | **int32** | StopTimeout is a timeout between the container&#x27;s stop signal being sent and SIGKILL being sent. If not provided, the default will be used. If 0 is used, stop signal will not be sent, and SIGKILL will be sent instead. Optional. | [optional] [default to null]
**Sysctl** | **map[string]string** | Sysctl sets kernel parameters for the container | [optional] [default to null]
**Systemd** | **string** | Systemd is whether the container will be started in systemd mode. Valid options are \&quot;true\&quot;, \&quot;false\&quot;, and \&quot;always\&quot;. \&quot;true\&quot; enables this mode only if the binary run in the container is sbin/init or systemd. \&quot;always\&quot; unconditionally enables systemd mode. \&quot;false\&quot; unconditionally disables systemd mode. If enabled, mounts and stop signal will be modified. If set to \&quot;always\&quot; or set to \&quot;true\&quot; and conditionally triggered, conflicts with StopSignal. If not specified, \&quot;false\&quot; will be assumed. Optional. | [optional] [default to null]
**Terminal** | **bool** | Terminal is whether the container will create a PTY. Optional. | [optional] [default to null]
**Timezone** | **string** | Timezone is the timezone inside the container. Local means it has the same timezone as the host machine | [optional] [default to null]
**Utsns** | [***Namespace**](Namespace.md) |  | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

