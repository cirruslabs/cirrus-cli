// +build linux darwin windows

package heuristic

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"net"
)

func getCloudBuildSubnet(ctx context.Context) string {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return ""
	}
	defer cli.Close()

	network, err := cli.NetworkInspect(ctx, CloudBuildNetworkName, types.NetworkInspectOptions{})
	if err != nil {
		return ""
	}

	if len(network.IPAM.Config) != 1 {
		return ""
	}

	return network.IPAM.Config[0].Subnet
}

func GetCloudBuildIP(ctx context.Context) string {
	// Are we running in Cloud Build?
	cloudBuildSubnet := getCloudBuildSubnet(ctx)
	if cloudBuildSubnet == "" {
		return ""
	}

	_, cloudBuildNetwork, err := net.ParseCIDR(cloudBuildSubnet)
	if err != nil {
		return ""
	}

	// Pick up first IP address of the interface attached to the Cloud Build network
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}

	for _, addr := range addrs {
		interfaceIP, _, err := net.ParseCIDR(addr.String())
		if err != nil {
			continue
		}

		if cloudBuildNetwork.Contains(interfaceIP) {
			return interfaceIP.String()
		}
	}

	return ""
}
