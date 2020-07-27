package testutil

import (
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/golang/protobuf/proto" //nolint:staticcheck // https://github.com/cirruslabs/cirrus-ci-agent/issues/14
	"testing"
)

func GetBasicContainerInstance(t *testing.T, image string) *api.Task_Instance {
	instancePayload, err := proto.Marshal(&api.ContainerInstance{
		Image: image,
	})
	if err != nil {
		t.Fatal(err)
	}

	return &api.Task_Instance{
		Type:    "container",
		Payload: instancePayload,
	}
}
