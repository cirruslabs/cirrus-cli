# PortMapping

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ContainerPort** | **int32** | ContainerPort is the port number that will be exposed from the container. Mandatory. | [optional] [default to null]
**HostIp** | **string** | HostIP is the IP that we will bind to on the host. If unset, assumed to be 0.0.0.0 (all interfaces). | [optional] [default to null]
**HostPort** | **int32** | HostPort is the port number that will be forwarded from the host into the container. If omitted, a random port on the host (guaranteed to be over 1024) will be assigned. | [optional] [default to null]
**Protocol** | **string** | Protocol is the protocol forward. Must be either \&quot;tcp\&quot;, \&quot;udp\&quot;, and \&quot;sctp\&quot;, or some combination of these separated by commas. If unset, assumed to be TCP. | [optional] [default to null]
**Range_** | **int32** | Range is the number of ports that will be forwarded, starting at HostPort and ContainerPort and counting up. This is 1-indexed, so 1 is assumed to be a single port (only the Hostport:Containerport mapping will be added), 2 is two ports (both Hostport:Containerport and Hostport+1:Containerport+1), etc. If unset, assumed to be 1 (a single port). Both hostport + range and containerport + range must be less than 65536. | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

