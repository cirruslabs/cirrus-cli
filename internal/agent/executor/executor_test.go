//go:build linux

package executor_test

import (
	"context"
	"github.com/cirruslabs/cirrus-cli/internal/agent/client"
	"github.com/cirruslabs/cirrus-cli/internal/agent/executor"
	"github.com/cirruslabs/cirrus-cli/internal/agent/testutil"
	"github.com/cirruslabs/cirrus-cli/pkg/api"
	"github.com/google/uuid"
	vault "github.com/hashicorp/vault/api"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"testing"
)

// TestVaultSpecificVariableExpansion ensures that:
//
//  1. We expand most of the environment variables before Vault-boxed variable resolution.
//
//     This allows us to compose CIRRUS_VAULT_URL and VAULT[...] out of other variables
//     (see CIRRUS_VAULT_URL and VAULT_BOXED_VALUE for examples).
//
//  2. We delay expanding the variables that point to Vault-boxed variables until all
//     Vault-boxed variables are expanded.
//
//     This allows us to compose variables out of other Vault-boxed variables,
//     (see VALUE_USING_VAULT_BOXED_VALUE for an example).
func TestVaultSpecificVariableExpansion(t *testing.T) {
	ctx := context.Background()

	var vaultToken = uuid.New().String()

	// Create and start the Vault container
	request := testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        testutil.VaultContainerImage,
			ExposedPorts: []string{"8200/tcp"},
			Env: map[string]string{
				"VAULT_DEV_ROOT_TOKEN_ID": vaultToken,
			},
		},
		Started: true,
	}
	container, err := testcontainers.GenericContainer(ctx, request)
	require.NoError(t, err)
	defer container.Terminate(ctx)

	// Create demo data
	vaultURL, err := container.Endpoint(ctx, "http")
	require.NoError(t, err)

	vaultClient, err := vault.NewClient(vault.DefaultConfig())
	require.NoError(t, err)

	require.NoError(t, vaultClient.SetAddress(vaultURL))
	vaultClient.SetToken(vaultToken)

	_, err = vaultClient.KVv2("secret").Put(ctx, "account", map[string]interface{}{
		"password": "not really a secret",
	})
	require.NoError(t, err)

	// Initialize Cirrus CI service mock
	cirrusCIServiceMock := testutil.NewCirrusCIServiceMock(t,
		&api.CommandsResponse{
			TimeoutInSeconds: 60,
			Environment: map[string]string{
				"INDIRECT_VAULT_URL":                   vaultURL,
				"CIRRUS_VAULT_URL":                     "${INDIRECT_VAULT_URL}",
				"CIRRUS_VAULT_TOKEN":                   vaultToken,
				"PASSWORD_KEY":                         "password",
				"INDIRECT_VAULT_BOXED_VALUE":           "VAULT[secret/data/account data.$PASSWORD_KEY]",
				"VAULT_BOXED_VALUE":                    "${INDIRECT_VAULT_BOXED_VALUE}",
				"VALUE_USING_VAULT_BOXED_VALUE":        "${VAULT_BOXED_VALUE}",
				"VALUE_USING_VAULT_BOXED_VALUE_PREFIX": "prefix-${VAULT_BOXED_VALUE}",
				"VALUE_USING_VAULT_BOXED_VALUE_SUFFIX": "${VAULT_BOXED_VALUE}-suffix",
				"VALUE_USING_VAULT_BOXED_VALUE_BOTH":   "prefix-${VAULT_BOXED_VALUE}-suffix",
			},
			Commands: []*api.Command{
				{
					Name: "test",
					Instruction: &api.Command_ScriptInstruction{
						ScriptInstruction: &api.ScriptInstruction{
							Scripts: []string{
								"test \"$VALUE_USING_VAULT_BOXED_VALUE\" = \"not really a secret\"",
								"test \"$VALUE_USING_VAULT_BOXED_VALUE_PREFIX\" = \"prefix-not really a secret\"",
								"test \"$VALUE_USING_VAULT_BOXED_VALUE_SUFFIX\" = \"not really a secret-suffix\"",
								"test \"$VALUE_USING_VAULT_BOXED_VALUE_BOTH\" = \"prefix-not really a secret-suffix\"",
							},
						},
					},
				},
			},
		},
	)

	// Initialize Cirrus CI service client
	conn, err := grpc.Dial(cirrusCIServiceMock.Address(),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)

	client.InitClient(conn)

	// Run the executor
	executor := executor.NewExecutor(0, "", "", "", "",
		t.TempDir())
	executor.RunBuild(context.Background())

	// Make sure that the commands succeeded
	require.True(t, cirrusCIServiceMock.Finished(), "task has not finished")
	require.True(t, cirrusCIServiceMock.Succeeded(), "task has not succeeded")
}

func TestLimitCommands(t *testing.T) {
	commands := []*api.Command{
		{Name: "a"},
		{Name: "b"},
		{Name: "c"},
		{Name: "d"},
	}

	examples := []struct {
		Description      string
		FromName, ToName string
		Expected         []*api.Command
	}{
		{"unspecified bounds", "", "", commands},
		{"zero bound (beginning)", "a", "a", []*api.Command{}},
		{"zero bound (middle)", "b", "b", []*api.Command{}},
		{"zero bound (ending)", "d", "d", []*api.Command{}},
		{"zero bound (unspecified beginning)", "", "a", []*api.Command{}},
		{"only from (beginning)", "a", "", commands},
		{"only from (middle)", "b", "", []*api.Command{
			{Name: "b"},
			{Name: "c"},
			{Name: "d"},
		}},
		{"only from (ending)", "d", "", []*api.Command{
			{Name: "d"},
		}},
		{"only to (beginning)", "", "b", []*api.Command{
			{Name: "a"},
		}},
		{"only to (middle)", "", "c", []*api.Command{
			{Name: "a"},
			{Name: "b"},
		}},
		{"only to (ending)", "", "d", []*api.Command{
			{Name: "a"},
			{Name: "b"},
			{Name: "c"},
		}},
		{"nonexistent", "X", "Y", commands},
	}

	for _, example := range examples {
		t.Run(example.Description, func(t *testing.T) {
			require.Equal(t, example.Expected, executor.BoundedCommands(commands, example.FromName, example.ToName))
		})
	}
}
