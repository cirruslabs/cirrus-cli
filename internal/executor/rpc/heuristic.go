package rpc

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

func getDockerBridgeIP(ctx context.Context) string {
	// Worst-case scenario, but still better than nothing
	// since there's still a chance this would work with
	// a Docker daemon configured by default.
	const assumedBridgeIP = "172.17.0.1"

	iface, err := net.InterfaceByName(getDockerBridgeInterface(ctx))
	if err != nil {
		return assumedBridgeIP
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return assumedBridgeIP
	}

	if len(addrs) != 0 {
		ip, _, err := net.ParseCIDR(addrs[0].String())
		if err != nil {
			return assumedBridgeIP
		}

		return ip.String()
	}

	return assumedBridgeIP
}
