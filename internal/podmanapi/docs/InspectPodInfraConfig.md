# InspectPodInfraConfig

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**DNSOption** | **[]string** | DNSOption is a set of DNS options that will be used by the infra container&#x27;s resolv.conf and shared with the remainder of the pod. | [optional] [default to null]
**DNSSearch** | **[]string** | DNSSearch is a set of DNS search domains that will be used by the infra container&#x27;s resolv.conf and shared with the remainder of the pod. | [optional] [default to null]
**DNSServer** | **[]string** | DNSServer is a set of DNS Servers that will be used by the infra container&#x27;s resolv.conf and shared with the remainder of the pod. | [optional] [default to null]
**HostAdd** | **[]string** | HostAdd adds a number of hosts to the infra container&#x27;s resolv.conf which will be shared with the rest of the pod. | [optional] [default to null]
**HostNetwork** | **bool** | HostNetwork is whether the infra container (and thus the whole pod) will use the host&#x27;s network and not create a network namespace. | [optional] [default to null]
**NetworkOptions** | [**map[string][]string**](array.md) | NetworkOptions are additional options for each network | [optional] [default to null]
**Networks** | **[]string** | Networks is a list of CNI networks the pod will join. | [optional] [default to null]
**NoManageHosts** | **bool** | NoManageHosts indicates that the pod will not manage /etc/hosts and instead each container will handle their own. | [optional] [default to null]
**NoManageResolvConf** | **bool** | NoManageResolvConf indicates that the pod will not manage resolv.conf and instead each container will handle their own. | [optional] [default to null]
**PortBindings** | [**map[string][]InspectHostPort**](array.md) | PortBindings are ports that will be forwarded to the infra container and then shared with the pod. | [optional] [default to null]
**StaticIP** | [***[]int32**](array.md) |  | [optional] [default to null]
**StaticMAC** | [***[]int32**](array.md) |  | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

