package parallels

import (
	"context"
	"encoding/json"
)

func SharedNetworkHostIP(ctx context.Context) (string, error) {
	stdout, _, err := Prlsrvctl(ctx, "net", "info", "Shared", "-j")
	if err != nil {
		return "", err
	}

	type AdapterInfo struct {
		IPv4Address string `json:"IPv4 address"`
	}

	type NetInfo struct {
		ParallelsAdapter AdapterInfo `json:"Parallels adapter"`
	}

	var netInfo NetInfo

	if err := json.Unmarshal([]byte(stdout), &netInfo); err != nil {
		return "", err
	}

	return netInfo.ParallelsAdapter.IPv4Address, nil
}
