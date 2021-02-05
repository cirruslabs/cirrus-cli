package executor

import (
	"context"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance"
	"github.com/cirruslabs/cirrus-cli/internal/executor/options"
	"github.com/golang/protobuf/ptypes"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDockerfileImageTemplate(t *testing.T) {
	anyInstance, err := ptypes.MarshalAny(&api.PrebuiltImageInstance{
		Repository: "cirrus-ci-community/d41d8cd98f00b204e9800998ecf8427e",
		Reference:  "latest",
	})
	if err != nil {
		t.Fatal(err)
	}

	tasks := []*api.Task{
		{
			Name:     "TestDockerfileImageTemplate",
			Instance: anyInstance,
		},
	}

	containerOpts := options.ContainerOptions{
		DockerfileImageTemplate: "gcr.io/cirrus-ci-community/%s:latest",
	}

	e, err := New(context.Background(), ".", tasks, WithContainerOptions(containerOpts))
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "gcr.io/cirrus-ci-community/d41d8cd98f00b204e9800998ecf8427e:latest",
		e.build.GetTask(0).Instance.(*instance.PrebuiltInstance).Image)
}
