package agent_test

import (
	"context"
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/executor/agent"
	"testing"
)

func TestGetAgentVolume(t *testing.T) {
	volumeName, err := agent.GetAgentVolume(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(volumeName, err)
}
