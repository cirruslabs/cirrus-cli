package testutil

import (
	"context"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor"
	"github.com/cirruslabs/cirrus-cli/pkg/parser"
	"github.com/golang/protobuf/proto" //nolint:staticcheck // https://github.com/cirruslabs/cirrus-ci-agent/issues/14
	"github.com/stretchr/testify/require"
	"path/filepath"
	"testing"
)

func GetBasicContainerInstance(t *testing.T, image string) *api.Task_Instance {
	instancePayload, err := proto.Marshal(&api.ContainerInstance{
		Image: image,
	})
	if err != nil {
		t.Fatal(err)
	}

	return &api.Task_Instance{
		Type:    "container",
		Payload: instancePayload,
	}
}

func Execute(t *testing.T, projectDir string) {
	dir := TempDirPopulatedWith(t, projectDir)

	p := parser.Parser{}
	result, err := p.ParseFromFile(filepath.Join(dir, ".cirrus.yml"))
	if err != nil {
		t.Fatal(err)
	}

	require.Empty(t, result.Errors)
	require.NotEmpty(t, result.Tasks)

	e, err := executor.New(dir, result.Tasks)
	if err != nil {
		t.Fatal(err)
	}

	if err := e.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
}
