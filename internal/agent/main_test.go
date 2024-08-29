package agent

import (
	"context"
	"github.com/cirruslabs/cirrus-cli/internal/agent/client"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/metadata"
	"testing"
)

func Test_DialHTTPS(t *testing.T) {
	assert.Nil(t, checkEndpoint("https://grpc.cirrus-ci.com:443"))
}

func Test_DialNoSchema(t *testing.T) {
	assert.Nil(t, checkEndpoint("grpc.cirrus-ci.com:443"))
}

func checkEndpoint(endpoint string) error {
	clientConn, err := dialWithTimeout(context.Background(), endpoint, metadata.New(map[string]string{}))
	if err != nil {
		return err
	}

	defer clientConn.Close()

	client.InitClient(clientConn, &api.TaskIdentification{})

	return err
}
