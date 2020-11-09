# ContainerNetworkConfig

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**CniNetworks** | **[]string** | CNINetworks is a list of CNI networks to join the container to. If this list is empty, the default CNI network will be joined instead. If at least one entry is present, we will not join the default network (unless it is part of this list). Only available if NetNS is set to bridge. Optional. | [optional] [default to null]
**DnsOption** | **[]string** | DNSOptions is a set of DNS options that will be used in the container&#x27;s resolv.conf, replacing the host&#x27;s DNS options which are used by default. Conflicts with UseImageResolvConf. Optional. | [optional] [default to null]
**DnsSearch** | **[]string** | DNSSearch is a set of DNS search domains that will be used in the container&#x27;s resolv.conf, replacing the host&#x27;s DNS search domains which are used by default. Conflicts with UseImageResolvConf. Optional. | [optional] [default to null]
**DnsServer** | [**[][]int32**](array.md) | DNSServers is a set of DNS servers that will be used in the container&#x27;s resolv.conf, replacing the host&#x27;s DNS Servers which are used by default. Conflicts with UseImageResolvConf. Optional. | [optional] [default to null]
**Expose** | [***interface{}**](interface{}.md) | Expose is a number of ports that will be forwarded to the container if PublishExposedPorts is set. Expose is a map of uint16 (port number) to a string representing protocol. Allowed protocols are \&quot;tcp\&quot;, \&quot;udp\&quot;, and \&quot;sctp\&quot;, or some combination of the three separated by commas. If protocol is set to \&quot;\&quot; we will assume TCP. Only available if NetNS is set to Bridge or Slirp, and PublishExposedPorts is set. Optional. | [optional] [default to null]
**Hostadd** | **[]string** | HostAdd is a set of hosts which will be added to the container&#x27;s etc/hosts file. Conflicts with UseImageHosts. Optional. | [optional] [default to null]
**Netns** | [***Namespace**](Namespace.md) |  | [optional] [default to null]
**NetworkOptions** | [**map[string][]string**](array.md) | NetworkOptions are additional options for each network Optional. | [optional] [default to null]
**Portmappings** | [**[]PortMapping**](PortMapping.md) | PortBindings is a set of ports to map into the container. Only available if NetNS is set to bridge or slirp. Optional. | [optional] [default to null]
**PublishImagePorts** | **bool** | PublishExposedPorts will publish ports specified in the image to random unused ports (guaranteed to be above 1024) on the host. This is based on ports set in Expose below, and any ports specified by the Image (if one is given). Only available if NetNS is set to Bridge or Slirp. | [optional] [default to null]
**StaticIp** | [***[]int32**](array.md) |  | [optional] [default to null]
**StaticIpv6** | [***[]int32**](array.md) |  | [optional] [default to null]
**StaticMac** | [***[]int32**](array.md) |  | [optional] [default to null]
**UseImageHosts** | **bool** | UseImageHosts indicates that /etc/hosts should not be managed by Podman, and instead sourced from the image. Conflicts with HostAdd. | [optional] [default to null]
**UseImageResolveConf** | **bool** | UseImageResolvConf indicates that resolv.conf should not be managed by Podman, but instead sourced from the image. Conflicts with DNSServer, DNSSearch, DNSOption. | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

