package abstractcontainer

import (
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"google.golang.org/protobuf/proto"
)

type IsolationContainer struct {
	Proto *api.PersistentWorkerInstance
}

func (isolationContainer *IsolationContainer) Dockerfile() string {
	return isolationContainer.Proto.Isolation.GetContainer().Dockerfile
}

func (isolationContainer *IsolationContainer) DockerArguments() map[string]string {
	return isolationContainer.Proto.Isolation.GetContainer().DockerArguments
}

func (isolationContainer *IsolationContainer) Platform() api.Platform {
	return isolationContainer.Proto.Isolation.GetContainer().Platform
}

func (isolationContainer *IsolationContainer) SetImage(image string) {
	isolationContainer.Proto.Isolation.GetContainer().Image = image
}

func (isolationContainer *IsolationContainer) Message() proto.Message {
	return isolationContainer.Proto
}
