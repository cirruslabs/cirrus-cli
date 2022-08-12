package docker

import (
	"encoding/base64"
	"encoding/json"
	"github.com/docker/cli/cli/config"
	"io"
	"strings"
)

func XRegistryAuthForImage(reference string) (string, error) {
	dockerConfig := config.LoadDefaultConfigFile(io.Discard)
	referenceParts := strings.SplitN(reference, "/", 2)

	authConfig, err := dockerConfig.GetAuthConfig(referenceParts[0])
	if err != nil {
		return "", err
	}

	authConfigJSON, err := json.Marshal(&authConfig)
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(authConfigJSON), nil
}
