# InspectAdditionalNetwork

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**AdditionalMACAddresses** | **[]string** | AdditionalMacAddresses is a set of additional MAC Addresses beyond the first. CNI may configure more than one interface for a single network, which can cause this. | [optional] [default to null]
**DriverOpts** | **map[string]string** | DriverOpts is presently unused and maintained exclusively for compatibility. | [optional] [default to null]
**EndpointID** | **string** | EndpointID is unused, maintained exclusively for compatibility. | [optional] [default to null]
**Gateway** | **string** | Gateway is the IP address of the gateway this network will use. | [optional] [default to null]
**GlobalIPv6Address** | **string** | GlobalIPv6Address is the global-scope IPv6 Address for this network. | [optional] [default to null]
**GlobalIPv6PrefixLen** | **int64** | GlobalIPv6PrefixLen is the length of the subnet mask of this network. | [optional] [default to null]
**IPAMConfig** | **map[string]string** | IPAMConfig is presently unused and maintained exclusively for compatibility. | [optional] [default to null]
**IPAddress** | **string** | IPAddress is the IP address for this network. | [optional] [default to null]
**IPPrefixLen** | **int64** | IPPrefixLen is the length of the subnet mask of this network. | [optional] [default to null]
**IPv6Gateway** | **string** | IPv6Gateway is the IPv6 gateway this network will use. | [optional] [default to null]
**Links** | **[]string** | Links is presently unused and maintained exclusively for compatibility. | [optional] [default to null]
**MacAddress** | **string** | MacAddress is the MAC address for the interface in this network. | [optional] [default to null]
**NetworkID** | **string** | Name of the network we&#x27;re connecting to. | [optional] [default to null]
**SecondaryIPAddresses** | **[]string** | SecondaryIPAddresses is a list of extra IP Addresses that the container has been assigned in this network. | [optional] [default to null]
**SecondaryIPv6Addresses** | **[]string** | SecondaryIPv6Addresses is a list of extra IPv6 Addresses that the container has been assigned in this networ. | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

