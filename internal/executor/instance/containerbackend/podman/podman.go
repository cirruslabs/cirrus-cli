package podman

import (
	"encoding/base64"
	"encoding/json"
	"github.com/containers/image/v5/pkg/docker/config"
	"github.com/containers/image/v5/types"
)

func XRegistryAuthForImage(reference string) (string, error) {
	authConfig, err := config.GetCredentials(&types.SystemContext{}, reference)
	if err != nil {
		return "", err
	}

	authConfigJSON, err := json.Marshal(&authConfig)
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(authConfigJSON), nil
}
