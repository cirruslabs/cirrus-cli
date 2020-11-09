# PodSpecGenerator

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**CgroupParent** | **string** | CgroupParent is the parent for the CGroup that the pod will create. This pod cgroup will, in turn, be the default cgroup parent for all containers in the pod. Optional. | [optional] [default to null]
**CniNetworks** | **[]string** | CNINetworks is a list of CNI networks that the infra container will join. As, by default, containers share their network with the infra container, these networks will effectively be joined by the entire pod. Only available when NetNS is set to Bridge, the default for root. Optional. | [optional] [default to null]
**DnsOption** | **[]string** | DNSOption is a set of DNS options that will be used in the infra container&#x27;s resolv.conf, which will, by default, be shared with all containers in the pod. Conflicts with NoInfra&#x3D;true. Optional. | [optional] [default to null]
**DnsSearch** | **[]string** | DNSSearch is a set of DNS search domains that will be used in the infra container&#x27;s resolv.conf, which will, by default, be shared with all containers in the pod. If not provided, DNS search domains from the host&#x27;s resolv.conf will be used. Conflicts with NoInfra&#x3D;true. Optional. | [optional] [default to null]
**DnsServer** | [**[][]int32**](array.md) | DNSServer is a set of DNS servers that will be used in the infra container&#x27;s resolv.conf, which will, by default, be shared with all containers in the pod. If not provided, the host&#x27;s DNS servers will be used, unless the only server set is a localhost address. As the container cannot connect to the host&#x27;s localhost, a default server will instead be set. Conflicts with NoInfra&#x3D;true. Optional. | [optional] [default to null]
**Hostadd** | **[]string** | HostAdd is a set of hosts that will be added to the infra container&#x27;s etc/hosts that will, by default, be shared with all containers in the pod. Conflicts with NoInfra&#x3D;true and NoManageHosts. Optional. | [optional] [default to null]
**Hostname** | **string** | Hostname is the pod&#x27;s hostname. If not set, the name of the pod will be used (if a name was not provided here, the name auto-generated for the pod will be used). This will be used by the infra container and all containers in the pod as long as the UTS namespace is shared. Optional. | [optional] [default to null]
**InfraCommand** | **[]string** | InfraCommand sets the command that will be used to start the infra container. If not set, the default set in the Libpod configuration file will be used. Conflicts with NoInfra&#x3D;true. Optional. | [optional] [default to null]
**InfraConmonPidFile** | **string** | InfraConmonPidFile is a custom path to store the infra container&#x27;s conmon PID. | [optional] [default to null]
**InfraImage** | **string** | InfraImage is the image that will be used for the infra container. If not set, the default set in the Libpod configuration file will be used. Conflicts with NoInfra&#x3D;true. Optional. | [optional] [default to null]
**Labels** | **map[string]string** | Labels are key-value pairs that are used to add metadata to pods. Optional. | [optional] [default to null]
**Name** | **string** | Name is the name of the pod. If not provided, a name will be generated when the pod is created. Optional. | [optional] [default to null]
**Netns** | [***Namespace**](Namespace.md) |  | [optional] [default to null]
**NetworkOptions** | [**map[string][]string**](array.md) | NetworkOptions are additional options for each network Optional. | [optional] [default to null]
**NoInfra** | **bool** | NoInfra tells the pod not to create an infra container. If this is done, many networking-related options will become unavailable. Conflicts with setting any options in PodNetworkConfig, and the InfraCommand and InfraImages in this struct. Optional. | [optional] [default to null]
**NoManageHosts** | **bool** | NoManageHosts indicates that /etc/hosts should not be managed by the pod. Instead, each container will create a separate /etc/hosts as they would if not in a pod. Conflicts with HostAdd. | [optional] [default to null]
**NoManageResolvConf** | **bool** | NoManageResolvConf indicates that /etc/resolv.conf should not be managed by the pod. Instead, each container will create and manage a separate resolv.conf as if they had not joined a pod. Conflicts with NoInfra&#x3D;true and DNSServer, DNSSearch, DNSOption. Optional. | [optional] [default to null]
**PodCreateCommand** | **[]string** | PodCreateCommand is the command used to create this pod. This will be shown in the output of Inspect() on the pod, and may also be used by some tools that wish to recreate the pod (e.g. &#x60;podman generate systemd --new&#x60;). Optional. | [optional] [default to null]
**Portmappings** | [**[]PortMapping**](PortMapping.md) | PortMappings is a set of ports to map into the infra container. As, by default, containers share their network with the infra container, this will forward the ports to the entire pod. Only available if NetNS is set to Bridge or Slirp. Optional. | [optional] [default to null]
**SharedNamespaces** | **[]string** | SharedNamespaces instructs the pod to share a set of namespaces. Shared namespaces will be joined (by default) by every container which joins the pod. If not set and NoInfra is false, the pod will set a default set of namespaces to share. Conflicts with NoInfra&#x3D;true. Optional. | [optional] [default to null]
**StaticIp** | [***[]int32**](array.md) |  | [optional] [default to null]
**StaticMac** | [***[]int32**](array.md) |  | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

