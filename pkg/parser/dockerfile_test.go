package parser_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"github.com/cirruslabs/cirrus-cli/pkg/larker/fs/memory"
	"github.com/cirruslabs/cirrus-cli/pkg/parser"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCalculateDockerfileHash(t *testing.T) {
	dockerfile := `FROM scratch

COPY docker docker
`
	script := `#!/bin/bash
`

	fs, err := memory.New(map[string][]byte{
		"docker/linux":     []byte(dockerfile),
		"docker/script.sh": []byte(script),
	})
	require.NoError(t, err)

	h := sha256.New()
	// Obligatory Dockerfile hash
	h.Write([]byte(dockerfile))
	// COPY docker docker
	h.Write([]byte(dockerfile))
	h.Write([]byte(script))
	expectedDockerfileHash := hex.EncodeToString(h.Sum([]byte{}))

	result, err := parser.New(parser.WithFileSystem(fs)).Parse(context.Background(), `task:
  container:
    dockerfile: docker/linux
`)
	require.NoError(t, err)
	require.Equal(t, expectedDockerfileHash,
		result.Tasks[0].Metadata.Properties["dockerfile_hash"])
}
