# ContainerSecurityConfig

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ApparmorProfile** | **string** | ApparmorProfile is the name of the Apparmor profile the container will use. Optional. | [optional] [default to null]
**CapAdd** | **[]string** | CapAdd are capabilities which will be added to the container. Conflicts with Privileged. Optional. | [optional] [default to null]
**CapDrop** | **[]string** | CapDrop are capabilities which will be removed from the container. Conflicts with Privileged. Optional. | [optional] [default to null]
**Groups** | **[]string** | Groups are a list of supplemental groups the container&#x27;s user will be granted access to. Optional. | [optional] [default to null]
**Idmappings** | [***IdMappingOptions**](IDMappingOptions.md) |  | [optional] [default to null]
**NoNewPrivileges** | **bool** | NoNewPrivileges is whether the container will set the no new privileges flag on create, which disables gaining additional privileges (e.g. via setuid) in the container. | [optional] [default to null]
**Privileged** | **bool** | Privileged is whether the container is privileged. Privileged does the following: Adds all devices on the system to the container. Adds all capabilities to the container. Disables Seccomp, SELinux, and Apparmor confinement. (Though SELinux can be manually re-enabled). TODO: this conflicts with things. TODO: this does more. | [optional] [default to null]
**ProcfsOpts** | **[]string** | ProcOpts are the options used for the proc mount. | [optional] [default to null]
**ReadOnlyFilesystem** | **bool** | ReadOnlyFilesystem indicates that everything will be mounted as read-only | [optional] [default to null]
**SeccompPolicy** | **string** | SeccompPolicy determines which seccomp profile gets applied the container. valid values: empty,default,image | [optional] [default to null]
**SeccompProfilePath** | **string** | SeccompProfilePath is the path to a JSON file containing the container&#x27;s Seccomp profile. If not specified, no Seccomp profile will be used. Optional. | [optional] [default to null]
**SelinuxOpts** | **[]string** | SelinuxProcessLabel is the process label the container will use. If SELinux is enabled and this is not specified, a label will be automatically generated if not specified. Optional. | [optional] [default to null]
**Umask** | **string** | Umask is the umask the init process of the container will be run with. | [optional] [default to null]
**User** | **string** | User is the user the container will be run as. Can be given as a UID or a username; if a username, it will be resolved within the container, using the container&#x27;s /etc/passwd. If unset, the container will be run as root. Optional. | [optional] [default to null]
**Userns** | [***Namespace**](Namespace.md) |  | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

