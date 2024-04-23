package abstractcontainer

import (
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"google.golang.org/protobuf/proto"
)

type ContainerInstance struct {
	Proto *api.ContainerInstance
}

func (containerInstance *ContainerInstance) Dockerfile() string {
	return containerInstance.Proto.Dockerfile
}

func (containerInstance *ContainerInstance) DockerArguments() map[string]string {
	return containerInstance.Proto.DockerArguments
}

func (containerInstance *ContainerInstance) Platform() api.Platform {
	return containerInstance.Proto.Platform
}

func (containerInstance *ContainerInstance) SetImage(image string) {
	containerInstance.Proto.Image = image
}

func (containerInstance *ContainerInstance) Message() proto.Message {
	return containerInstance.Proto
}
