package vaultunboxer

import (
	"context"
	"fmt"

	vault "github.com/hashicorp/vault/api"
)

type JWTAuth struct {
	Token string
	Role  string
	Path  string
}

func (jwtAuth *JWTAuth) Login(ctx context.Context, client *vault.Client) (*vault.Secret, error) {
	data := map[string]interface{}{
		"jwt": jwtAuth.Token,
	}

	if jwtAuth.Role != "" {
		data["role"] = jwtAuth.Role
	}

	if jwtAuth.Path == "" {
		jwtAuth.Path = "jwt"
	}

	return client.Logical().WriteWithContext(ctx, fmt.Sprintf("auth/%s/login", jwtAuth.Path), data)
}
