package testutil

import (
	"context"
	"github.com/cirruslabs/cirrus-ci-agent/api"
	"github.com/cirruslabs/cirrus-cli/internal/executor"
	"github.com/cirruslabs/cirrus-cli/pkg/parser"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
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

func Execute(t *testing.T, dir string) {
	p := parser.Parser{}
	result, err := p.ParseFromFile(filepath.Join(dir, ".cirrus.yml"))
	if err != nil {
		t.Fatal(err)
	}

	require.Empty(t, result.Errors)
	require.NotEmpty(t, result.Tasks)

	// Create a logger with the maximum verbosity
	logger := logrus.New()
	logger.Level = logrus.TraceLevel

	e, err := executor.New(dir, result.Tasks, executor.WithLogger(logger))
	if err != nil {
		t.Fatal(err)
	}

	if err := e.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
}
