package cirruscimock

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/docker/go-connections/nat"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"testing"
)

func S3Client(t *testing.T) *s3.Client {
	t.Helper()

	ctx := context.Background()

	localstackContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "localstack/localstack",
			WaitingFor:   wait.ForHTTP("/_localstack/health").WithPort("4566/tcp"),
			ExposedPorts: []string{"4566/tcp"},
		},
		Started: true,
	})
	require.NoError(t, err)

	exposedPort, err := nat.NewPort("tcp", "4566")
	require.NoError(t, err)

	mappedPort, err := localstackContainer.MappedPort(ctx, exposedPort)
	require.NoError(t, err)

	endpoint := fmt.Sprintf("http://s3.localhost.localstack.cloud:%d", mappedPort.Int())

	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("id", "secret", "")),
	)
	require.NoError(t, err)

	return s3.NewFromConfig(cfg, func(options *s3.Options) {
		options.BaseEndpoint = aws.String(endpoint)
		options.UsePathStyle = true
	})
}
