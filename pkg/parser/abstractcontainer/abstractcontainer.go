package abstractcontainer

import (
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"google.golang.org/protobuf/proto"
)

type AbstractContainer interface {
	Dockerfile() string
	DockerArguments() map[string]string
	Platform() api.Platform

	SetImage(image string)

	Message() proto.Message
}
