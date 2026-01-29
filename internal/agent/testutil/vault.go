package testutil

import (
	"context"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func StartVaultContainer(ctx context.Context, vaultToken string) (testcontainers.Container, error) {
	request := testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        VaultContainerImage,
			ExposedPorts: []string{"8200/tcp"},
			Env: map[string]string{
				"VAULT_DEV_ROOT_TOKEN_ID":  vaultToken,
				"VAULT_DEV_LISTEN_ADDRESS": "0.0.0.0:8200",
			},
			WaitingFor: wait.ForHTTP("/v1/sys/health").
				WithPort("8200/tcp").
				WithStartupTimeout(60 * time.Second),
		},
		Started: true,
	}

	return testcontainers.GenericContainer(ctx, request)
}
