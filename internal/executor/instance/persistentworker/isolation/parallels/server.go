package parallels

import (
	"context"
	"encoding/json"
	"fmt"
)

type ServerInfo struct {
	VMHome string `json:"VM home"`
}

func GetServerInfo(ctx context.Context) (*ServerInfo, error) {
	stdout, stderr, err := Prlsrvctl(ctx, "info", "--json")
	if err != nil {
		return nil, fmt.Errorf("%w: failed to retrieve Parallels server info: %q",
			ErrVMFailed, firstNonEmptyLine(stderr))
	}

	var serverInfo ServerInfo

	if err := json.Unmarshal([]byte(stdout), &serverInfo); err != nil {
		return nil, fmt.Errorf("%w: failed to decode Parallels server info: %v",
			ErrVMFailed, err)
	}

	return &serverInfo, nil
}
