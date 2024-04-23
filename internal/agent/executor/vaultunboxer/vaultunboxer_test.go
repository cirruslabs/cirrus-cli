//go:build linux

package vaultunboxer_test

import (
	"context"
	"github.com/cirruslabs/cirrus-cli/internal/agent/executor/vaultunboxer"
	"github.com/cirruslabs/cirrus-cli/internal/agent/testutil"
	"github.com/google/uuid"
	vault "github.com/hashicorp/vault/api"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"testing"
	"time"
)

func TestVault(t *testing.T) {
	ctx := context.Background()

	var vaultToken = uuid.New().String()

	// Create and start the HashiCorp's Vault container
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

	client, err := vault.NewClient(vault.DefaultConfig())
	require.NoError(t, err)

	require.NoError(t, client.SetAddress(vaultURL))
	client.SetToken(vaultToken)

	const (
		secretKeyValue = "secret key value"
	)

	_, err = client.KVv2("secret").Put(ctx, "keys", map[string]interface{}{
		"admin": secretKeyValue,
	})
	require.NoError(t, err)

	// Unbox a Vault-boxed value
	selector, err := vaultunboxer.NewBoxedValue("VAULT[secret/data/keys data.admin]")
	require.NoError(t, err)

	selector_arg, err := vaultunboxer.NewBoxedValue("VAULT[secret/data/keys data.admin version=1]")
	require.NoError(t, err)

	secretValue, err := vaultunboxer.New(client).Unbox(ctx, selector)
	require.NoError(t, err)
	require.Equal(t, secretKeyValue, secretValue)

	secretValue, err = vaultunboxer.New(client).Unbox(ctx, selector_arg)
	require.NoError(t, err)
	require.Equal(t, secretKeyValue, secretValue)
}

func TestVaultUseCache(t *testing.T) {
	ctx := context.Background()

	var vaultToken = uuid.New().String()

	// Create and start the HashiCorp's Vault container
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

	// Create demo data with an OTP token
	vaultURL, err := container.Endpoint(ctx, "http")
	require.NoError(t, err)

	client, err := vault.NewClient(vault.DefaultConfig())
	require.NoError(t, err)

	require.NoError(t, client.SetAddress(vaultURL))
	client.SetToken(vaultToken)

	require.NoError(t, client.Sys().MountWithContext(ctx, "totp", &vault.MountInput{Type: "totp"}))

	_, err = client.Logical().WriteWithContext(ctx, "/totp/keys/test", map[string]interface{}{
		"key":    "Y64VEVMBTSXCYIWRSHRNDZW62MPGVU2G",
		"period": "1",
	})
	require.NoError(t, err)

	// Prepare
	vaultUnboxer := vaultunboxer.New(client)

	otpWithCache, err := vaultunboxer.NewBoxedValue("VAULT[totp/code/test code]")
	require.NoError(t, err)

	otpWithoutCache, err := vaultunboxer.NewBoxedValue("VAULT_NOCACHE[totp/code/test code]")
	require.NoError(t, err)

	// Make sure that VAULT[...] works when no cache entry is found
	first, err := vaultUnboxer.Unbox(ctx, otpWithCache)
	require.NoError(t, err)

	// Wait for the OTP code to rotate
	time.Sleep(time.Second * 5)

	// Make sure that VAULT[...] uses cache
	second, err := vaultUnboxer.Unbox(ctx, otpWithCache)
	require.NoError(t, err)

	require.EqualValues(t, first, second)

	// Make sure that VAULT_NOCACHE[...] busts the cache
	third, err := vaultUnboxer.Unbox(ctx, otpWithoutCache)
	require.NoError(t, err)

	require.NotEqualValues(t, second, third)

	// Make sure out assumption about OTP code rotating every 5 seconds is valid
	time.Sleep(time.Second * 5)

	fourth, err := vaultUnboxer.Unbox(ctx, otpWithoutCache)
	require.NoError(t, err)

	require.NotEqualValues(t, third, fourth)

	// Make sure that VAULT[...] invocation re-uses the value previously retrieved by VAULT_NOCACHE[...] invocation
	fifth, err := vaultUnboxer.Unbox(ctx, otpWithCache)
	require.NoError(t, err)

	require.EqualValues(t, fourth, fifth)
}

func TestVaultDictionaryAsJSON(t *testing.T) {
	ctx := context.Background()

	var vaultToken = uuid.New().String()

	// Create and start the HashiCorp's Vault container
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

	client, err := vault.NewClient(vault.DefaultConfig())
	require.NoError(t, err)

	require.NoError(t, client.SetAddress(vaultURL))
	client.SetToken(vaultToken)

	_, err = client.KVv2("secret").Put(ctx, "token", map[string]interface{}{
		"json_token": map[string]interface{}{
			"secret1": "secret1",
			"secret2": "secret2",
		},
	})
	require.NoError(t, err)

	// Unbox a Vault-boxed value
	selector, err := vaultunboxer.NewBoxedValue("VAULT[secret/data/token data.json_token]")
	require.NoError(t, err)

	secretValue, err := vaultunboxer.New(client).Unbox(ctx, selector)
	require.NoError(t, err)
	require.Equal(t, "{\"secret1\":\"secret1\",\"secret2\":\"secret2\"}", secretValue)
}
