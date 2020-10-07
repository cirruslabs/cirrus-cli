package heuristic

import (
	"context"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"net"
)

func getDockerBridgeInterface(ctx context.Context) string {
	const assumedBridgeInterface = "docker0"

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return assumedBridgeInterface
	}
	defer cli.Close()

	network, err := cli.NetworkInspect(ctx, "bridge", types.NetworkInspectOptions{})
	if err != nil {
		return assumedBridgeInterface
	}

	bridgeInterface, ok := network.Options["com.docker.network.bridge.name"]
	if !ok {
		return assumedBridgeInterface
	}

	return bridgeInterface
}

func GetDockerBridgeIP(ctx context.Context) string {
	iface, err := net.InterfaceByName(getDockerBridgeInterface(ctx))
	if err != nil {
		return ""
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return ""
	}

	if len(addrs) != 0 {
		ip, _, err := net.ParseCIDR(addrs[0].String())
		if err != nil {
			return ""
		}

		return ip.String()
	}

	return ""
}

func getCloudBuildSubnet(ctx context.Context) string {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return ""
	}
	defer cli.Close()

	// https://cloud.google.com/cloud-build/docs/build-config#network
	network, err := cli.NetworkInspect(ctx, "cloudbuild", types.NetworkInspectOptions{})
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
		interfaceIP, interfaceNetwork, err := net.ParseCIDR(addr.String())
		if err != nil {
			continue
		}

		if interfaceNetwork == cloudBuildNetwork {
			return interfaceIP.String()
		}
	}

	return ""
}
