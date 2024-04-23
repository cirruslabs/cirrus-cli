package executor

import (
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance"
	"github.com/cirruslabs/cirrus-cli/internal/executor/options"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/anypb"
	"testing"
)

func TestDockerfileImageTemplate(t *testing.T) {
	anyInstance, err := anypb.New(&api.PrebuiltImageInstance{
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

	e, err := New(".", tasks, WithContainerOptions(containerOpts))
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "gcr.io/cirrus-ci-community/d41d8cd98f00b204e9800998ecf8427e:latest",
		e.build.GetTask(0).Instance.(*instance.PrebuiltInstance).Image)
}
