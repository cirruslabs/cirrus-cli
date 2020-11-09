# ContainerStorageConfig

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Devices** | [**[]LinuxDevice**](LinuxDevice.md) | Devices are devices that will be added to the container. Optional. | [optional] [default to null]
**Image** | **string** | Image is the image the container will be based on. The image will be used as the container&#x27;s root filesystem, and its environment vars, volumes, and other configuration will be applied to the container. Conflicts with Rootfs. At least one of Image or Rootfs must be specified. | [optional] [default to null]
**ImageVolumeMode** | **string** | ImageVolumeMode indicates how image volumes will be created. Supported modes are \&quot;ignore\&quot; (do not create), \&quot;tmpfs\&quot; (create as tmpfs), and \&quot;anonymous\&quot; (create as anonymous volumes). The default if unset is anonymous. Optional. | [optional] [default to null]
**Init** | **bool** | Init specifies that an init binary will be mounted into the container, and will be used as PID1. | [optional] [default to null]
**InitPath** | **string** | InitPath specifies the path to the init binary that will be added if Init is specified above. If not specified, the default set in the Libpod config will be used. Ignored if Init above is not set. Optional. | [optional] [default to null]
**Ipcns** | [***Namespace**](Namespace.md) |  | [optional] [default to null]
**Mounts** | [**[]Mount**](Mount.md) | Mounts are mounts that will be added to the container. These will supersede Image Volumes and VolumesFrom volumes where there are conflicts. Optional. | [optional] [default to null]
**OverlayVolumes** | [**[]OverlayVolume**](OverlayVolume.md) | Overlay volumes are named volumes that will be added to the container. Optional. | [optional] [default to null]
**Rootfs** | **string** | Rootfs is the path to a directory that will be used as the container&#x27;s root filesystem. No modification will be made to the directory, it will be directly mounted into the container as root. Conflicts with Image. At least one of Image or Rootfs must be specified. | [optional] [default to null]
**RootfsPropagation** | **string** | RootfsPropagation is the rootfs propagation mode for the container. If not set, the default of rslave will be used. Optional. | [optional] [default to null]
**ShmSize** | **int64** | ShmSize is the size of the tmpfs to mount in at /dev/shm, in bytes. Conflicts with ShmSize if IpcNS is not private. Optional. | [optional] [default to null]
**Volumes** | [**[]NamedVolume**](NamedVolume.md) | Volumes are named volumes that will be added to the container. These will supersede Image Volumes and VolumesFrom volumes where there are conflicts. Optional. | [optional] [default to null]
**VolumesFrom** | **[]string** | VolumesFrom is a set of containers whose volumes will be added to this container. The name or ID of the container must be provided, and may optionally be followed by a : and then one or more comma-separated options. Valid options are &#x27;ro&#x27;, &#x27;rw&#x27;, and &#x27;z&#x27;. Options will be used for all volumes sourced from the container. | [optional] [default to null]
**WorkDir** | **string** | WorkDir is the container&#x27;s working directory. If unset, the default, /, will be used. Optional. | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

