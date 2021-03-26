package runconfig_test

import (
	"fmt"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/runconfig"
	"github.com/cirruslabs/cirrus-cli/internal/executor/platform"
	"github.com/stretchr/testify/assert"
	"math"
	"testing"
)

func TestSetAgentVersionWithoutDowngrade(t *testing.T) {
	rc := &runconfig.RunConfig{}

	// No downgrade
	prettyLowVersion := "0.1.0"
	rc.SetAgentVersionWithoutDowngrade(prettyLowVersion)
	assert.Equal(t, platform.DefaultAgentVersion, rc.GetAgentVersion())

	// Only upgrade
	prettyHighVersion := fmt.Sprintf("%d.0.0", math.MaxInt32)
	rc.SetAgentVersionWithoutDowngrade(prettyHighVersion)
	assert.Equal(t, prettyHighVersion, rc.GetAgentVersion())
}
