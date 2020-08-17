package testutil

import (
	"context"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor"
	"github.com/cirruslabs/cirrus-cli/pkg/parser"
	"github.com/cirruslabs/echelon"
	"github.com/cirruslabs/echelon/renderers"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"os"
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

func Execute(t *testing.T, dir string) error {
	// Create a logger with the maximum verbosity
	renderer := renderers.NewSimpleRenderer(os.Stdout, nil)
	logger := echelon.NewLogger(echelon.TraceLevel, renderer)

	return ExecuteWithOptions(t, dir, executor.WithLogger(logger))
}

func ExecuteWithOptions(t *testing.T, dir string, opts ...executor.Option) error {
	p := parser.Parser{}
	result, err := p.ParseFromFile(filepath.Join(dir, ".cirrus.yml"))
	if err != nil {
		t.Fatal(err)
	}

	require.Empty(t, result.Errors)
	require.NotEmpty(t, result.Tasks)

	e, err := executor.New(dir, result.Tasks, opts...)
	if err != nil {
		t.Fatal(err)
	}

	return e.Run(context.Background())
}
