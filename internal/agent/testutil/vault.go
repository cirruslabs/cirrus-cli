package testutil

import (
	"context"
	"strings"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func StartVaultContainer(ctx context.Context, vaultToken string) (testcontainers.Container, error) {
	request := testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        VaultContainerImage,
			Entrypoint:   []string{"vault"},
			Cmd:          []string{"server", "-config=/tmp/vault.hcl", "-dev", "-dev-root-token-id=" + vaultToken, "-dev-listen-address=0.0.0.0:8200"},
			ExposedPorts: []string{"8200/tcp"},
			Files: []testcontainers.ContainerFile{
				{
					Reader:            strings.NewReader("disable_mlock = true\n"),
					ContainerFilePath: "/tmp/vault.hcl",
					FileMode:          0o644,
				},
			},
			WaitingFor: wait.ForHTTP("/v1/sys/health").
				WithPort("8200/tcp").
				WithStartupTimeout(60 * time.Second),
		},
		Started: true,
	}

	return testcontainers.GenericContainer(ctx, request)
}
